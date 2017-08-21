#!/usr/bin/env python3

import argparse
import io
import json
import os.path as path
import sys
from abc import abstractmethod
from typing import NamedTuple, Dict, Tuple, List, Set, Iterable, Any, Union

import svgwrite

FILE_TYPES = ['svg']
COLOR_RULES = ['goroutine', 'function', 'module']
LAYOUTS = ['goroutines', 'funccalls']
CELL_WIDTH = 20
CELL_HEIGHT = 20


class UncorrespondingRecord(ValueError):
    pass


class Unreachable(Exception):
    pass


Frame = NamedTuple('Frame', (
    ('PC', int),
    ('Func', dict),
    ('Function', str),
    ('File', str),
    ('Line', int),
    ('Entry', int),
))


class RunFunc(NamedTuple('_BaseRunFunc', (
        ('tag', str),
        ('n', int),
        ('frames', Tuple[Frame, ...]),
        ('gid', int),
))):
    def __new__(cls, *args, **kwargs):
        return super().__new__(cls, *args, **kwargs)

    @property
    def callee(self):
        return self.frames[0]

    @property
    def caller(self):
        return self.frames[1:]


class Func(NamedTuple('_Func', (
        ('frames', Tuple[Frame, ...]),
        ('start_time', int),
        ('end_time', Union[int, None]),
        ('gid', int),
))):
    def __new__(cls, *args, **kwargs):
        return super().__new__(cls, *args, **kwargs)

    @property
    def callee(self):
        return self.frames[0]

    @property
    def caller(self):
        return self.frames[1:]

    def is_alive(self, n):
        return self.start_time <= n <= self.end_time


class GoRoutine(NamedTuple('_BaseGoRoutine', (
        ('func', Func),
        ('start_time', int),
        ('end_time', int),
        ('parent', Any),  # actually types is Union[GoRoutine, None]
        ('gid', int),
))):
    def __new__(cls, *args, **kwargs):
        return super().__new__(cls, *args, **kwargs)

    @property
    def stack(self):
        return self.func.caller

    @property
    def parents(self):
        count = 0
        gr = self
        while gr.parent is not None:
            count += 1
            gr = gr.parent
        return count

    @classmethod
    def make_from(cls, f: Func):
        return GoRoutine(
            func=f,
            start_time=f.start_time,
            end_time=f.end_time,
            parent=None,
            gid=f.gid,
        )

    def is_alive(self, f: Func) -> bool:
        # end_timeはNoneである可能性あり
        cond = tuple(i is None for i in (f.end_time, self.end_time))

        if cond == (True, True):
            return self.start_time < f.start_time
        elif cond == (False, False):
            return self.start_time < f.start_time < f.end_time < self.end_time
        elif cond == (False, True):
            return self.start_time < f.start_time < f.end_time
        elif cond == (True, False):
            return False

        raise Unreachable("bug", cond)

    def is_caller(self, callee: Func):
        return self.gid == callee.gid

    def to_call(self, f: Func):
        """
        GoRoutineのcallstackにfを積み、fを呼び出したもととして扱います。
        元のオブジェクトは更新されません。

        :param f: 呼び出し対象の関数
        :return: fを呼び出した後のGoRoutine
        """
        if self.gid != f.gid:
            raise ValueError('not matched gid (goroutine id)')

        return GoRoutine(
            func=f,
            start_time=f.start_time,
            end_time=f.end_time,
            parent=self,
            gid=f.gid,
        )

    def to_return(self):
        """
        callstackのトップから関数を1つ取り出し、関数の実行を終了させた扱いをする。
        元のオブジェクトは更新されません。
        全ての関数の実行が終了していたら、Noneを返す。

        :return: 関数実行終了後のGoRoutine または None
        """
        return self.parent


def read_records(file: io.TextIOWrapper) -> Iterable[Tuple[int, Dict]]:
    for n, line in enumerate(file):
        line = line.rstrip()
        if not line:
            continue
        data = json.loads(line)
        yield n, data


def parse_logs(file: io.TextIOWrapper) -> Iterable[RunFunc]:
    """
    ログを1行1行パースする。

    :param file:
    :return:
    """
    for n, record in read_records(file):
        tag = record['tag']
        list_frames = record['frames']  # type: List[Dict]
        # hashableでないdictは削除する
        for f in list_frames:
            f['Func'] = None

        frames = tuple(Frame(**f) for f in list_frames)

        yield RunFunc(
            tag=tag,
            n=n,
            frames=frames,
            gid=record['gid'],
        )


def find_start_end_pairs(records: Iterable[RunFunc]) -> Iterable[Func]:
    """
    ログから、関数の開始・終了のペアを探し出す。
    関数が終了していない場合、end=Noneを返す。

    :param records:
    :return:
    """
    runfuncs = []  # type: List[RunFunc]

    for rec in records:
        if rec.tag == 'funcStart':
            runfuncs.append(rec)

        elif rec.tag == 'funcEnd':
            for start in runfuncs:
                if start.gid != rec.gid: continue
                if start.caller != rec.caller: continue
                if start.callee.Function != rec.callee.Function: continue

                runfuncs.remove(start)
                yield Func(
                    frames=start.frames,
                    start_time=start.n,
                    end_time=rec.n,
                    gid=start.gid,
                )
                break
            else:
                raise UncorrespondingRecord('un-corresponding record', rec)

    # 終了していない関数
    for rec in runfuncs:
        yield Func(
            frames=rec.frames,
            start_time=rec.n,
            end_time=None,
            gid=rec.gid,
        )


def sort_by_start_time(records: Iterable[Func]) -> Iterable[Func]:
    return sorted(records, key=lambda f: f.start_time)


def build_callstack(funcs: Iterable[Func]) -> Tuple[List[int], List[List[GoRoutine]]]:
    max_depth = []  # type: List[int]
    gr_history = []  # type: List[List[GoRoutine]]

    gidmap = {}  # type: Dict[int, int]
    goroutines = []  # type: List[GoRoutine]
    depth = []  # type: List[int]
    for f in funcs:
        for i, gr in enumerate(goroutines):
            while (gr is not None) and (gr.end_time is not None) and (gr.end_time < f.start_time):
                # forcible exit of running functions.
                goroutines[i] = gr = gr.to_return()
                depth[i] -= 1
                continue
            if gr is None: continue

            if gr.is_caller(f):
                # gr is probably caller of f(). assumed to call f() on gr.
                goroutines[i] = gr.to_call(f)
                depth[i] += 1
                max_depth[i] = max(max_depth[i], depth[i])
                break
        else:
            # create a goroutine
            gr = GoRoutine.make_from(f)

            if gr.gid in gidmap.keys():
                i = gidmap[gr.gid]
                goroutines[i] = gr
                depth[i] = 1
            else:
                goroutines.append(gr)
                depth.append(1)
                max_depth.append(1)
                gidmap[gr.gid] = len(goroutines) - 1

        # yield of current goroutine status
        gr_history.append(list(goroutines))

    return max_depth, gr_history


def generate_colors(ncolors: int, s: float, v: float) -> List[str]:
    """
    HSV色相環上から等間隔にncolors色選び、HEX形式(#RRGGBB)の文字列として返す。
    円錐モデルを使用しているため、s <= vを満たさなければならない。

    :param ncolors: 色数
    :param s: 彩度 [0.0, v]
    :param v: 明度 [0.0, 1.0]
    :return: RGBカラーの配列
    """
    if not (ncolors >= 0):
        raise ValueError('ncolors must be not less than 0. but ncolors={}'.format(ncolors))
    if not (s <= v):
        raise ValueError('s <= v is not satisfied. s={}, v={}'.format(s, v))

    def hsv2rgb(h: float) -> Tuple[int, int, int]:
        """
        convert color space from HSV to RGB

        :param h: 色相 [0, 1.0]
        :return:
        """
        hh = 360 * h / 60
        c = s
        x = c * (1 - abs((hh % 2) - 1))
        r, g, b = (
            (c, x, 0),
            (x, c, 0),
            (0, c, x),
            (0, x, c),
            (x, 0, c),
            (c, 0, x),
        )[int(hh)]  # type: Tuple[float, float, float]
        return (
            int(255 * (v - c + r)),
            int(255 * (v - c + g)),
            int(255 * (v - c + b)),
        )

    def colorstr(rgb: Tuple[int, int, int]) -> str:
        return '#%02x%02x%02x' % rgb

    # colors[i] = color
    colors = []  # type: List[str]
    for i in range(ncolors):
        h = i / ncolors
        colors.append(colorstr(hsv2rgb(h)))

    return colors


class ColorRule:
    def __init__(self, ncolors: int, s: float, v: float):
        self._colors = generate_colors(ncolors, s, v)

    def _get_by_hash(self, hashable: object):
        return self._colors[hash(hashable) % len(self._colors)]

    @abstractmethod
    def get(self, index: int = None, goroutine: GoRoutine = None): pass


class GoroutineColorRule(ColorRule):
    def get(self, index: int = None, goroutine: GoRoutine = None):
        if index is not None:
            return self._get_by_hash(int(index))
        raise ValueError('index must be not None')


class FuncColorRule(ColorRule):
    def get(self, index: int = None, goroutine: GoRoutine = None):
        if goroutine is not None:
            return self._get_by_hash(goroutine.func.callee.Function)
        raise ValueError('goroutine must be not None')


class ModuleColorRule(ColorRule):
    def get(self, index: int = None, goroutine: GoRoutine = None):
        if goroutine is not None:
            f = goroutine.func.callee.Function  # type: str
            pkg = f[f.rfind('/'):]
            return self._get_by_hash(pkg)
        raise ValueError('goroutine must be not None')


def to_svg(max_depth: List[int], gr_history: List[List[GoRoutine]], color: ColorRule, layout: str,
           output: io.TextIOBase):
    dwg = svgwrite.Drawing()

    if layout == 'goroutines':
        number_of_gr = len(gr_history[-1])
        start_time = []  # type: List[int]
        end_time = []  # type: List[int]
        goroutines = []  # type: List[GoRoutine]

        for i in range(number_of_gr):
            for gr in gr_history:
                if len(gr) <= i: continue
                if gr[i] is None: continue
                start_time.append(gr[i].start_time)
                end_time.append(gr[i].end_time)
                goroutines.append(gr[i])
                break
            else:
                raise Unreachable('bug')

        last_time = max(filter(lambda x: x is not None, end_time))
        for i, start, end, gr in zip(range(number_of_gr), start_time, end_time, goroutines):
            if end is None:
                width = last_time - start
            else:
                width = end - start

            dwg.add(dwg.rect(
                insert=(start * CELL_WIDTH, i * CELL_HEIGHT),
                size=(width * CELL_WIDTH, 1 * CELL_HEIGHT),
                fill=color.get(index=i, goroutine=None)
            ))
            callee = gr.func.callee
            dwg.add(dwg.text(
                '{} ({}:{})'.format(
                    callee.Function, path.basename(callee.File), callee.Line
                ),
                insert=(start * CELL_WIDTH, (i + 1) * CELL_HEIGHT),
                font_size=CELL_HEIGHT,
                fill='#000',
            ))

        dwg.viewbox(
            minx=0, miny=0,
            width=last_time * CELL_WIDTH, height=number_of_gr * CELL_HEIGHT,
        )
    elif layout == 'funccalls':
        endless_goroutines = {}  # type: Dict[Tuple[int, int], GoRoutine]
        rendered_goroutines = set()  # type: Set[GoRoutine]
        last_time = -1

        for goroutines in gr_history:
            offset_y = 0
            for i, gr in enumerate(goroutines):
                y = offset_y
                offset_y += max_depth[i] + 1
                if gr is None: continue
                if gr in rendered_goroutines: continue
                if gr.end_time is None:
                    if (i, gr.start_time) not in endless_goroutines:
                        endless_goroutines[(i, gr.start_time)] = gr
                    continue

                last_time = max(last_time, gr.end_time)
                width = gr.end_time - gr.start_time
                dwg.add(dwg.rect(
                    insert=(gr.start_time * CELL_WIDTH, (y + gr.parents) * CELL_HEIGHT),
                    size=(width * CELL_WIDTH, CELL_HEIGHT),
                    fill=color.get(index=i, goroutine=gr),
                ))
                callee = gr.func.callee
                dwg.add(dwg.text(
                    '{} ({}:{})'.format(
                        callee.Function, path.basename(callee.File), callee.Line
                    ),
                    insert=(gr.start_time * CELL_WIDTH, (y + gr.parents + 1) * CELL_HEIGHT),
                    font_size=CELL_HEIGHT,
                    fill='#000',
                ))
                rendered_goroutines.add(gr)

        # rendering of end-less functions
        for key, gr in endless_goroutines.items():
            i, _ = key
            y = sum(d + 1 for d in max_depth[:i])
            if gr is None: continue
            if gr.end_time is not None: continue

            width = last_time - gr.start_time
            dwg.add(dwg.rect(
                insert=(gr.start_time * CELL_WIDTH, (y + gr.parents) * CELL_HEIGHT),
                size=(width * CELL_WIDTH, CELL_HEIGHT),
                fill=color.get(index=i, goroutine=gr),
            ))
            callee = gr.func.callee
            dwg.add(dwg.text(
                '{} ({}:{})'.format(
                    callee.Function, path.basename(callee.File), callee.Line
                ),
                insert=(gr.start_time * CELL_WIDTH, (y + gr.parents + 1) * CELL_HEIGHT),
                font_size=CELL_HEIGHT,
                fill='#000',
            ))

        dwg.viewbox(
            minx=0, miny=0,
            width=last_time * CELL_WIDTH, height=sum(d + 1 for d in max_depth) * CELL_HEIGHT,
        )
    else:
        raise ValueError('Unknown layout type: {}'.format(layout))

    dwg.write(output)


def parse_args():
    parser = argparse.ArgumentParser(
        description='Simple log visualizer',
        epilog=None,
    )
    try:
        # Monkey patch for format collapsing issue.
        parser._get_formatter = lambda: argparse.HelpFormatter(
            prog=sys.argv[0],
            max_help_position=32)
    except:
        pass

    parser.add_argument('-i', '--input',
                        type=argparse.FileType('r'),
                        default=sys.stdin,
                        metavar='FILE',
                        help='read from FILE instead of stdin')
    parser.add_argument('-o', '--output',
                        type=argparse.FileType('w'),
                        default=sys.stdout,
                        metavar='FILE',
                        help='write to FILE instead of stdout')
    parser.add_argument('-t', '--type',
                        choices=FILE_TYPES,
                        default='svg',
                        metavar='TYPE',
                        help='change output file type\n(choices from {})'.format(
                            ', '.join("'{}'".format(ft) for ft in FILE_TYPES)
                        ))
    parser.add_argument('-c', '--color-rule',
                        choices=COLOR_RULES,
                        default='goroutine',
                        metavar='TYPE',
                        help='change coloring rule (default: coloring per goroutine)')
    parser.add_argument('-l', '--layout',
                        choices=LAYOUTS,
                        metavar='LAYOUT',
                        help='change layout (default: goroutines)')
    return parser.parse_args()


def main():
    try:
        args = parse_args()

        if args.color_rule == 'goroutine':
            color = GoroutineColorRule(5, 0.6, 0.7)
        elif args.color_rule == 'function':
            color = FuncColorRule(5, 0.6, 0.7)
        elif args.color_rule == 'module':
            color = FuncColorRule(5, 0.6, 0.7)
        else:
            raise ValueError('Invalid color-rule: {}'.format(args.color_rule))

        max_depth, gr_history = build_callstack(sort_by_start_time(find_start_end_pairs(parse_logs(args.input))))

        if args.type == 'svg':
            to_svg(max_depth, gr_history, color, args.layout, args.output)
        else:
            raise ValueError('Unsupported type: {}'.format(args.type))

        return 0
    except ValueError as e:
        print('ERROR', e.args, file=sys.stderr)
        return 1


if __name__ == '__main__':
    exit(main())
