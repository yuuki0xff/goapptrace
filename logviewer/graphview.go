package logviewer

import (
	"context"
	"fmt"
	"image"
	"log"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/marcusolsson/tui-go"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
	"golang.org/x/sync/errgroup"
	"golang.org/x/tools/container/intsets"
)

var (
	defaultScrollSpeed = image.Point{
		X: 10,
		Y: 10,
	}
)

type GraphState struct {
	State    GState
	Error    error
	Lines    []Line
	Selected logutil.FuncLogID

	ScrollMode ScrollMode
	// X軸方向のスクロール量
	OffsetX int
	// Y軸方向のスクロール量
	OffsetY int

	ScrollSpeed image.Point
}
type GraphStateMutable struct {
	GraphState
}

func (s *GraphStateMutable) UpdateOffset(dx, dy int) {
	s.OffsetX += dx
	s.OffsetY += dy

	if s.OffsetX > 0 {
		s.OffsetX = 0
	}
	if s.OffsetY > 0 {
		s.OffsetY = 0
	}
}

type GraphCache struct {
	FcList  []funcCallWithFuncIDs
	FsList  []restapi.GoLineInfo
	FList   []restapi.FuncInfo
	LogInfo restapi.LogStatus
	GMap    map[logutil.GID]restapi.Goroutine

	logID  string
	client restapi.ClientWithCtx
}

func (c *GraphCache) Update(logID string, client restapi.ClientWithCtx) error {
	c.logID = logID
	c.client = client

	var eg errgroup.Group
	eg.Go(func() error {
		var err error
		c.FcList, c.LogInfo, err = c.getFCLogs()
		return err
	})
	eg.Go(func() error {
		var err error
		c.GMap, err = c.getGoroutines()
		return err
	})
	return eg.Wait()
}

// getFCLogs returns latest function call logs.
func (c *GraphCache) getFCLogs() ([]funcCallWithFuncIDs, restapi.LogStatus, error) {
	ch2 := make(chan funcCallWithFuncIDs, 10000)
	var conf restapi.LogStatus

	var eg errgroup.Group
	eg.Go(func() error {
		ch, err := c.client.SearchFuncCalls(c.logID, restapi.SearchFuncCallParams{
			//Limit:     fetchRecords,
			SortKey:   restapi.SortByID,
			SortOrder: restapi.DescendingSortOrder,
		})
		if err != nil {
			return err
		}

		go func() {
			defer close(ch2)
			// TODO: 内部でAPI呼び出しを伴う遅い関数。必要に応じて並列度を上げる。
			c.withFuncIDs(ch, ch2)
		}()
		return nil
	})
	eg.Go(func() error {
		var err error
		conf, err = c.client.LogStatus(c.logID)
		return err
	})

	if err := eg.Wait(); err != nil {
		return nil, restapi.LogStatus{}, err
	}

	// 関数呼び出しのログの一覧
	fcList := make([]funcCallWithFuncIDs, 0, 10000)
	for item := range ch2 {
		// マスクされているイベントを削除する
		if item.isMasked(&conf.Metadata.UI) {
			continue
		}

		fcList = append(fcList, item)
	}
	return fcList, conf, nil
}
func (c *GraphCache) getGoroutines() (map[logutil.GID]restapi.Goroutine, error) {
	ch, err := c.client.Goroutines(c.logID)
	if err != nil {
		return nil, err
	}
	gm := make(map[logutil.GID]restapi.Goroutine, 10000)
	for g := range ch {
		gm[g.GID] = g
	}
	return gm, nil
}

// frames2funcs converts logutil.GoLineID to logutil.FuncID.
func (c *GraphCache) frames2funcs(frames []logutil.GoLineID) (funcs []logutil.FuncID) {
	for _, id := range frames {
		fs, err := c.client.GoLine(c.logID, strconv.Itoa(int(id)))
		if err != nil {
			log.Panic(err)
		}
		funcs = append(funcs, fs.Func)
	}
	return
}

// withFuncIDsはFuncCallのIDsを
func (c *GraphCache) withFuncIDs(in chan restapi.FuncCall, out chan funcCallWithFuncIDs) {
	for fc := range in {
		funcs := c.frames2funcs(fc.Frames)
		out <- funcCallWithFuncIDs{
			FuncCall: fc,
			funcs:    funcs,
		}
	}
}

// EndedFuncCallsは、時刻tの時点で実行が終了したFuncCallの数を返す。
func (c *GraphCache) EndedFuncCalls(t logutil.Time) int {
	n := 0
	for _, fc := range c.FcList {
		if fc.IsEnded() && fc.EndTime < t {
			n++
		}
	}
	return n
}

// RunningFuncCallsは、時刻tの時点で実行中のFuncCallの数を返す。
func (c *GraphCache) RunningFuncCalls(t logutil.Time) int {
	n := 0
	for _, fc := range c.FcList {
		if fc.StartTime < t {
			if !fc.IsEnded() || fc.EndTime >= t {
				n++
			}
		}
	}
	return n
}

type GraphVM struct {
	Root   Coordinator
	Client restapi.ClientWithCtx
	LogID  string

	m     sync.Mutex
	view  *GraphView
	state GraphStateMutable
	cache GraphCache
}

func (vm *GraphVM) UpdateInterval() time.Duration {
	return 0
}
func (vm *GraphVM) Update(ctx context.Context) {
	var cache GraphCache
	var lines []Line

	err := cache.Update(vm.LogID, vm.Client)
	if err == nil {
		lines = vm.buildLines(&cache)
	}

	vm.m.Lock()
	vm.view = nil
	vm.state.State = GWait
	vm.state.Error = err
	vm.state.Lines = lines
	vm.cache = cache
	vm.m.Unlock()

	vm.Root.NotifyVMUpdated()
}
func (vm *GraphVM) View() View {
	vm.m.Lock()
	defer vm.m.Unlock()

	if vm.view == nil {
		vm.view = &GraphView{
			VM:         vm,
			GraphState: vm.state.GraphState,
		}
	}
	return vm.view
}

// buildLinesは、graphを構成する線分を構築して返す。
func (vm *GraphVM) buildLines(c *GraphCache) (lines []Line) {
	vm.m.Lock()
	selected := vm.state.Selected
	vm.m.Unlock()

	fcList := c.FcList
	gMap := c.GMap

	// TODO: 活動していないgoroutineも表示する。goroutineが生きているのか、死んでいるのかを把握できない。

	// goroutineごとの、最も最初に活動のあった時刻に相当するX座標。
	// 関数呼び出し間のギャップ、つまりgoroutineが何も活動していない？と思われる区間を埋めるための線を描画するために使用する。
	firstXSet := make(map[logutil.GID]int, len(gMap))
	lastXSet := make(map[logutil.GID]int, len(gMap))

	var wg sync.WaitGroup
	wg.Add(2)

	// 長さとX座標を決める
	fcLen := make([]int, len(fcList))
	fcX := make([]int, len(fcList))
	go func() {
		defer wg.Done()
		// 関数の実行開始時刻が早い順(StartTimeの値が小さい順)にソートする。
		sort.Slice(fcList, func(i, j int) bool {
			return fcList[i].StartTime > fcList[j].StartTime
		})

		maxTime := logutil.Time(0)
		for _, fc := range fcList {
			if maxTime < fc.StartTime {
				maxTime = fc.StartTime
			}
			if maxTime < fc.EndTime {
				maxTime = fc.EndTime
			}
		}

		// 長さとX座標を決める
		calcXPos := func(t logutil.Time) int {
			return c.EndedFuncCalls(t)*2 + c.RunningFuncCalls(t)
		}
		for i, fc := range fcList {
			left := calcXPos(fc.StartTime)
			right := calcXPos(fc.EndTime)
			if !fc.IsEnded() {
				right = calcXPos(maxTime)
			}
			if left >= right {
				log.Panicf("bug: left=%d < right=%d: fc=%+v", left, right, fc)
			}
			fcX[i] = left
			fcLen[i] = right - left + 1
		}

		// 関数呼び出しのギャップを埋める線のX座標を計算する。
		for gid, g := range c.GMap {
			// firstXSetには、fcXのgoroutineごとの最小値を設定する。
			// lastXSetには、fcXのgoroutineごとの最大値を設定する。
			first := intsets.MaxInt
			last := intsets.MinInt
			exists := false

			for i, f := range fcList {
				if f.GID != gid {
					continue
				}
				exists = true
				if first > fcX[i] {
					first = fcX[i]
					if g.StartTime < f.StartTime {
						// goroutineは、関数fより少し早く開始された。
						// 開始位置を1つ前にする。
						first--
					}
				}
				x := fcX[i] + fcLen[i]
				if last < x {
					last = x
					if f.EndTime < g.EndTime {
						// goroutineは、関数fより少し遅く終了した。
						// 開始位置を1つ後ろにする。
						last++
					}
				}
			}
			if !exists {
				log.Panicf("not found function call logs of GID=%d", gid)
			}
			firstXSet[gid] = first
			lastXSet[gid] = last
		}
	}()

	// Y座標を決める
	gidY := make(map[logutil.GID]int, len(gMap))
	go func() {
		defer wg.Done()
		// 描画対象のGoroutine IDの小さい順にソートする。
		gidList := make([]logutil.GID, 0, len(gMap))
		for gid := range gMap {
			gidList = append(gidList, gid)
		}
		sort.Slice(gidList, func(i, j int) bool {
			return gidList[i] < gidList[j]
		})

		// GoroutineごとのY座標を決定する
		for idx, gid := range gidList {
			//log.Printf("GID=%d idx=%d", gid, idx)
			gidY[gid] = idx
		}
	}()

	lines = make([]Line, 0, len(fcList)+len(gMap))
	wg.Wait()

	// 関数呼び出し間のギャップを埋めるための線を追加。
	for gid := range gidY {
		length := lastXSet[gid] - firstXSet[gid]
		if length < 0 {
			log.Panicf("negative length: length = %d - %d = %d", lastXSet[gid], firstXSet[gid], length)
		}
		line := Line{
			Start: image.Point{
				X: firstXSet[gid],
				Y: gidY[gid],
			},
			Length:    length,
			Type:      HorizontalLine,
			StartDeco: LineTerminationNone,
			EndDeco:   LineTerminationNone,
			StyleName: "line.gap",
		}
		lines = append(lines, line)
	}

	for i := len(fcList) - 1; i >= 0; i-- {
		// fcListを逆順にループする。
		// 呼び出し元が呼び出し先のlineを上書きして見えなくしてしまうから。
		fc := fcList[i]

		// スタイル名の決定をする。
		styleName := "line."
		if fc.IsEnded() {
			styleName += "stopped"
		} else {
			styleName += "running"
		}
		if fc.ID == selected {
			styleName += ".selected"
		} else if fc.isPinned(&c.LogInfo.Metadata.UI) {
			styleName += ".marked"
		}

		// 水平線を追加
		line := Line{
			Start: image.Point{
				X: fcX[i],
				Y: gidY[fc.GID],
			},
			Length:    fcLen[i],
			Type:      HorizontalLine,
			StartDeco: LineTerminationNormal,
			EndDeco:   LineTerminationNormal,
			StyleName: styleName,
		}
		lines = append(lines, line)
	}
	return lines
}
func (vm *GraphVM) onGoback() {
	vm.Root.SetState(UIState{
		LogID: vm.LogID,
	})
}
func (vm *GraphVM) onChangedOffset(dx, dy int) {
	vm.m.Lock()
	vm.view = nil
	vm.state.UpdateOffset(dx, dy)
	vm.m.Unlock()

	vm.Root.NotifyVMUpdated()
}
func (vm *GraphVM) onChangedScrollMode(mode ScrollMode) {
	vm.m.Lock()
	vm.view = nil
	vm.state.ScrollMode = mode
	vm.m.Unlock()

	vm.Root.NotifyVMUpdated()
}

type GraphView struct {
	VM *GraphVM
	GraphState

	initOnce sync.Once
	widget   tui.Widget
	fc       tui.FocusChain

	graph       *GraphWidget
	graphScroll *ScrollWidget
}

func (v *GraphView) init() {
	switch v.State {
	case GLoading:
		space := tui.NewSpacer()
		v.widget = tui.NewVBox(
			space,
			v.newStatusBar(LoadingText),
		)
		v.fc = newFocusChain(space)
		return
	case GWait:
		if v.Error != nil {
			errmsg := newErrorMsg(v.Error)
			v.widget = tui.NewVBox(
				errmsg,
				tui.NewSpacer(),
				v.newStatusBar(ErrorText),
			)
			v.fc = newFocusChain(errmsg)
			return
		} else {
			var offsetMsg string

			v.graph = newGraphWidget()
			v.graph.SetLines(v.Lines)
			v.graphScroll = &ScrollWidget{
				ScrollableWidget: v.graph,
			}

			v.graphScroll.Scroll(v.OffsetX, v.OffsetY)
			switch v.ScrollMode {
			case ManualScrollMode:
				offsetMsg = fmt.Sprintf("%dx%d", v.OffsetX, v.OffsetY)
			case AutoScrollMode:
				v.graphScroll.AutoScroll(true, false)
			}

			v.widget = tui.NewVBox(
				v.graphScroll,
				v.newStatusBar(offsetMsg),
			)
			v.fc = newFocusChain(v.graph)
			return
		}
	default:
		log.Panic("bug")
	}
}
func (v *GraphView) Widget() tui.Widget {
	v.initOnce.Do(v.init)
	return v.widget
}
func (v *GraphView) Keybindings() map[string]func() {
	v.initOnce.Do(v.init)
	goback := func() {
		v.VM.onGoback()
	}
	up := func() {
		// AutoScrollModeのときでも、上下スクロールは可能にする。
		// そのため、このイベント発生時にはManualScrollModeに切り替えない。
		v.VM.onChangedOffset(0, v.scrollSpeed().Y)
	}
	right := func() {
		v.VM.onChangedScrollMode(ManualScrollMode)
		v.VM.onChangedOffset(-v.scrollSpeed().X, 0)
	}
	down := func() {
		// AutoScrollModeのときでも、上下スクロールは可能にする。
		// そのため、このイベント発生時にはManualScrollModeに切り替えない。
		v.VM.onChangedOffset(0, -v.scrollSpeed().Y)
	}
	left := func() {
		v.VM.onChangedScrollMode(ManualScrollMode)
		v.VM.onChangedOffset(v.scrollSpeed().X, 0)
	}
	autoScroll := func() {
		v.VM.onChangedScrollMode(AutoScrollMode)
	}

	return map[string]func(){
		"d":     goback,
		"k":     up,
		"Up":    up,
		"l":     right,
		"Right": right,
		"j":     down,
		"Down":  down,
		"h":     left,
		"Left":  left,
		// TODO: 原因を探る
		// WORKAROUND: tui-goが、shift+fをハンドリングできないみたい
		//"Shift+f":     autoScroll,
		"f": autoScroll,
	}
}
func (v *GraphView) FocusChain() tui.FocusChain {
	v.initOnce.Do(v.init)
	return v.fc
}

func (v *GraphView) scrollSpeed() image.Point {
	if v.ScrollSpeed.Eq(image.ZP) {
		return defaultScrollSpeed
	}
	return v.ScrollSpeed
}
func (v *GraphView) newStatusBar(text string) *tui.StatusBar {
	s := tui.NewStatusBar(LoadingText)
	s.SetPermanentText("Function Call History")
	s.SetText(text)
	return s
}
