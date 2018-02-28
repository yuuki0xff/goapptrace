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

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
	"github.com/yuuki0xff/tui-go"
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

	// X軸方向のスクロール量
	OffsetX int
	// Y軸方向のスクロール量
	OffsetY int
}
type GraphStateMutable struct {
	GraphState
}

func (s *GraphStateMutable) UpdateOffset(dx, dy int) {
	s.OffsetX += dx
	s.OffsetY += dy

	if s.OffsetX < 0 {
		s.OffsetX = 0
	}
	if s.OffsetY > 0 {
		s.OffsetY = 0
	}
}

type GraphCache struct {
	FcList  []funcCallWithFuncIDs
	FsList  []restapi.FuncStatusInfo
	FList   []restapi.FuncInfo
	LogInfo restapi.LogStatus
	GidSet  map[logutil.GID]bool
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
		c.FcList, c.GidSet, c.LogInfo, err = c.getFCLogs()
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
func (c *GraphCache) getFCLogs() ([]funcCallWithFuncIDs, map[logutil.GID]bool, restapi.LogStatus, error) {
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
		return nil, nil, restapi.LogStatus{}, err
	}

	// 関数呼び出しのログの一覧
	fcList := make([]funcCallWithFuncIDs, 0, 10000)
	// 存在するGIDの集合
	gidSet := make(map[logutil.GID]bool, 1000)
	for item := range ch2 {
		// マスクされているイベントを削除する
		if item.isMasked(&conf.Metadata.UI) {
			continue
		}

		fcList = append(fcList, item)
		gidSet[item.GID] = true
	}
	return fcList, gidSet, conf, nil
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

// frames2funcs converts logutil.FuncStatusID to logutil.FuncID.
func (c *GraphCache) frames2funcs(frames []logutil.FuncStatusID) (funcs []logutil.FuncID) {
	for _, id := range frames {
		fs, err := c.client.FuncStatus(c.logID, strconv.Itoa(int(id)))
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
		if fc.EndTime < t {
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
	gidSet := c.GidSet

	// TODO: 活動していないgoroutineも表示する。goroutineが生きているのか、死んでいるのかを把握できない。

	// goroutineごとの、最も最初に活動のあった時刻に相当するX座標。
	// 関数呼び出し間のギャップ、つまりgoroutineが何も活動していない？と思われる区間を埋めるための線を描画するために使用する。
	firstXSet := make(map[logutil.GID]int, len(gidSet))
	lastXSet := make(map[logutil.GID]int, len(gidSet))

	var wg sync.WaitGroup
	wg.Add(2)

	// 長さとX座標を決める
	fcLen := make([]int, len(fcList))
	fcX := make([]int, len(fcList))
	graphWidth := 0
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
			if graphWidth < last {
				graphWidth = last
			}
		}
	}()

	// Y座標を決める
	gidY := make(map[logutil.GID]int, len(gidSet))
	go func() {
		defer wg.Done()
		// 描画対象のGoroutine IDの小さい順にソートする。
		gidList := make([]logutil.GID, 0, len(gidSet))
		for gid := range gidSet {
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

	lines = make([]Line, 0, len(fcList)+len(gidSet))
	wg.Wait()

	// 関数呼び出し間のギャップを埋めるための線を追加。
	for gid := range gidY {
		length := lastXSet[gid] - firstXSet[gid]
		if length < 0 {
			log.Panicf("negative length: length = %d - %d = %d", lastXSet[gid], firstXSet[gid], length)
		}
		line := Line{
			Start: image.Point{
				X: -graphWidth + firstXSet[gid] + 1,
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
				X: -graphWidth + fcX[i] + 1,
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
	vm.state.UpdateOffset(dx, dy)
	vm.m.Unlock()

	vm.Root.NotifyVMUpdated()
}

type GraphView struct {
	VM *GraphVM
	GraphState

	initOnce sync.Once
	widget   tui.Widget
	fc       tui.FocusChain

	graph *GraphWidget
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
			v.graph = newGraphWidget()
			v.graph.SetLines(v.Lines)
			v.graph.SetOffset(image.Point{
				X: v.OffsetX,
				Y: v.OffsetY,
			})
			v.graph.SetOrigin(OriginTopRight)

			v.widget = tui.NewVBox(
				v.graph,
				v.newStatusBar(fmt.Sprintf("%dx%d", v.OffsetX, v.OffsetY)),
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
		v.VM.onChangedOffset(0, v.scrollSpeed().Y)
	}
	right := func() {
		v.VM.onChangedOffset(-v.scrollSpeed().X, 0)
	}
	down := func() {
		v.VM.onChangedOffset(0, -v.scrollSpeed().Y)
	}
	left := func() {
		v.VM.onChangedOffset(v.scrollSpeed().X, 0)
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
	}
}
func (v *GraphView) FocusChain() tui.FocusChain {
	v.initOnce.Do(v.init)
	return v.fc
}

//func (v *GraphView)
//func (v *GraphView)
//func (v *GraphView)
//func (v *GraphView)

func (v *GraphView) scrollSpeed() image.Point {
	if v.widget == nil {
		return defaultScrollSpeed
	}
	speed := v.widget.Size()
	// 一回のスクロールで、画面の5分の1くらいスクロールされる。
	speed.X /= 5
	speed.Y /= 5
	return speed
}
func (v *GraphView) newStatusBar(text string) *tui.StatusBar {
	s := tui.NewStatusBar(LoadingText)
	s.SetPermanentText("Function Call History")
	s.SetText(text)
	return s
}

type funcCallWithFuncIDs struct {
	restapi.FuncCall
	// 各フレームに対応するlogutil.FuncIDのリスト。
	// FuncStatusIDから変換するオーバーヘッドが大きいため、ここにキャッシュしておく。
	// TODO: FuncStatusID -> FuncIDをする共有キャッシュを作る
	funcs []logutil.FuncID
}

func (f *funcCallWithFuncIDs) isMasked(config *storage.UIConfig) (masked bool) {
	for _, fid := range f.funcs {
		if f, ok := config.Funcs[fid]; ok {
			masked = masked || f.Masked
		}
	}
	if g, ok := config.Goroutines[f.GID]; ok {
		masked = masked || g.Masked
	}
	return
}

func (f *funcCallWithFuncIDs) isPinned(config *storage.UIConfig) (pinned bool) {
	for _, fid := range f.funcs {
		if f, ok := config.Funcs[fid]; ok {
			pinned = pinned || f.Pinned
		}
	}
	if g, ok := config.Goroutines[f.GID]; ok {
		pinned = pinned || g.Pinned
	}
	return
}
