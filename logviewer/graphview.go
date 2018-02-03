package logviewer

import (
	"context"
	"fmt"
	"image"
	"log"
	"sort"
	"strconv"
	"sync"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
	"github.com/yuuki0xff/tui-go"
	"golang.org/x/sync/singleflight"
)

type GraphView struct {
	tui.Widget
	LogID string
	Root  *Controller

	running     uint32
	updateGroup singleflight.Group

	status *tui.StatusBar
	graph  *GraphWidget
	fc     tui.FocusChain

	// X軸方向のスクロール量
	offsetX int
	// Y軸方向のスクロール量
	offsetY int

	// FuncCallのリスト。
	// スクロールのたびに更新すると重たいので、キャッシュする。
	fcList []funcCallWithFuncIDs
	// 表示領域を確保しておくべきgoroutineの集合。
	gidSet    map[logutil.GID]bool
	logStatus restapi.LogStatus

	// 現在選択されている状態のFuncCallイベントのID
	selectedFLID logutil.FuncLogID
}

func newGraphView(logID string, root *Controller) *GraphView {
	v := &GraphView{
		LogID:  logID,
		Root:   root,
		status: tui.NewStatusBar(LoadingText),
		graph:  newGraphWidget(),
	}
	v.status.SetPermanentText("Function Call Graph")

	fc := &tui.SimpleFocusChain{}
	fc.Set(v)
	v.fc = fc
	v.Widget = tui.NewVBox(
		v.graph,
		v.status,
	)
	return v
}

func (v *GraphView) Update() {
	v.status.SetText(LoadingText)

	go v.updateGroup.Do("update", func() (interface{}, error) { // nolint: errcheck
		var err error
		v.Root.UI.Update(func() {
			if err != nil {
				//v.wrap.SetWidget(newErrorMsg(err))
				v.status.SetText(ErrorText)
			} else {
				//v.wrap.SetWidget(v.table)
				v.status.SetText(fmt.Sprintf("%dx%d", v.offsetX, v.offsetY))
			}
			lines := v.buildLines(v.graph.Size(), v.selectedFLID, &v.logStatus.Metadata.UI)
			v.graph.SetLines(lines)
		})
		return nil, nil
	})
}

// fcListおよびgidSetを更新する。
func (v *GraphView) updateCache() error {
	// TODO: ctxに対応する
	_, err, _ := v.updateGroup.Do("updateCache", func() (_ interface{}, err error) {
		var ch chan restapi.FuncCall
		ch, err = v.Root.Api.SearchFuncCalls(v.LogID, restapi.SearchFuncCallParams{
			//Limit:     fetchRecords,
			SortKey:   restapi.SortByID,
			SortOrder: restapi.DescendingSortOrder,
		})
		if err != nil {
			return
		}

		ch2 := make(chan funcCallWithFuncIDs, 10000)
		go func() {
			// TODO: 内部でAPI呼び出しを伴う遅い関数。必要に応じて並列度を上げる。
			v.withFuncIDs(ch, ch2)
			close(ch2)
		}()

		var conf restapi.LogStatus
		conf, err = v.Root.Api.LogStatus(v.LogID)
		if err != nil {
			return
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

		// TODO: 不要な再描画を省く
		// 排他制御を簡素化するためにUI.Update()を使用している。
		// なぜなら、UIのレンダリングはシングルスレッドで行われるため。
		v.Root.UI.Update(func() {
			v.fcList = fcList
			v.gidSet = gidSet
			v.logStatus = conf
		})
		// キャッシュを更新したので、画面に反映
		v.Update()
		return nil, nil
	})
	return err
}

func (v *GraphView) scrollSpeed() image.Point {
	speed := v.Size()
	// 一回のスクロールで、画面の5分の1くらいスクロールされる。
	speed.X /= 5
	speed.Y /= 5
	return speed
}

func (v *GraphView) SetKeybindings() {
	// TODO: スクロール処理を高速化する。
	//       現在は、サーバからのログ取得からレンダリングまでの全ての工程をスクロールのたびに行っている。
	//       描画済みの線のオフセットを変更するだけにすれば、軽量化出来るはず。
	gotoLogView := func() {
		v.Root.setView(newShowLogView(v.LogID, v.Root))
	}
	up := func() {
		v.offsetY += v.scrollSpeed().Y
		if v.offsetY > 0 {
			v.offsetY = 0
		}
		go v.Update()
	}
	right := func() {
		v.offsetX -= v.scrollSpeed().X
		if v.offsetX < 0 {
			v.offsetX = 0
		}
		go v.Update()
	}
	down := func() {
		v.offsetY -= v.scrollSpeed().Y
		go v.Update()
	}
	left := func() {
		v.offsetX += v.scrollSpeed().X
		go v.Update()
	}

	v.Root.UI.SetKeybinding("d", gotoLogView)
	v.Root.UI.SetKeybinding("k", up)
	v.Root.UI.SetKeybinding("Up", up)
	v.Root.UI.SetKeybinding("l", right)
	v.Root.UI.SetKeybinding("Right", right)
	v.Root.UI.SetKeybinding("j", down)
	v.Root.UI.SetKeybinding("Down", down)
	v.Root.UI.SetKeybinding("h", left)
	v.Root.UI.SetKeybinding("Left", left)
}
func (v *GraphView) FocusChain() tui.FocusChain {
	return v.fc
}
func (v *GraphView) Start(ctx context.Context) {
	update := func() {
		if err := v.updateCache(); err != nil {
			log.Println(err)
		}
	}
	go update()
	startAutoUpdateWorker(&v.running, ctx, update)
}

// buildLinesは、graphを構成する線分を構築して返す。
func (v *GraphView) buildLines(size image.Point, selectedFuncCall logutil.FuncLogID, config *storage.UIConfig) (lines []Line) {
	fcList := v.fcList
	gidSet := v.gidSet

	var wg sync.WaitGroup
	wg.Add(2)

	// 長さとX座標を決める
	fcLen := make([]int, len(fcList))
	fcX := make([]int, len(fcList))
	go func() {
		defer wg.Done()
		// 関数の実行終了時刻が遅い順(EndTimeの値が大きい順)にソートする。
		sort.Slice(fcList, func(i, j int) bool {
			return fcList[i].EndTime > fcList[j].EndTime
		})

		// FuncLogの子要素の一覧
		childMap := make(map[logutil.FuncLogID][]int, len(fcList))
		for i, item := range fcList {
			if item.ParentID == logutil.NotFoundParent {
				continue
			}
			childMap[item.ParentID] = append(childMap[item.ParentID], i)
		}

		// 長さを決める。
		// 長さは、実行中に生じたログの数+2。
		var length func(fc funcCallWithFuncIDs) int
		length = func(fc funcCallWithFuncIDs) int {
			childs, ok := childMap[fc.ID]
			if !ok {
				// 両端に終端記号を表示するため、長さは2にする。
				return 2
			}
			total := 0
			for _, idx := range childs {
				if fcLen[idx] == 0 {
					fcLen[idx] = length(fcList[idx])
				}
				total += fcLen[idx]
			}
			// 子要素の長さに、この要素の両端に終端記号を表示するための長さ(2)を足す。
			return total + 2
		}
		for i := range fcList {
			log.Printf("fcList[%d]: %+v", i, fcList[i])
			if fcLen[i] == 0 {
				fcLen[i] = length(fcList[i])
			}
		}

		// X座標を決める。
		// 最新のログは右側になるようにする。
		left := size.X
		for i := range fcList {
			fcX[i] = left - fcLen[i] + v.offsetX
			left--
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
			log.Printf("GID=%d idx=%d", gid, idx)
			gidY[gid] = idx + v.offsetY
		}
	}()

	lines = make([]Line, 0, len(fcList))
	wg.Wait()
	for i, fc := range fcList {
		if gidY[fc.GID] < 0 || gidY[fc.GID] >= size.Y {
			// 描画するのは水平線であるため、描画領域外の上下にある線は、絶対に描画されることはない。
			// そのため、無視する。
			continue
		}
		if fcX[i] >= size.X {
			// 描画領域外の右側にある線は描画されることはないため、無視する。
			continue
		}
		if fcX[i]+fcLen[i] < 0 {
			// 線の右端が描画領域の左側に達しない場合、この線は描画されることはないため、無視する。
			continue
		}

		// スタイル名の決定をする。
		styleName := "line."
		if fc.IsEnded() {
			styleName += "stopped"
		} else {
			styleName += "running"
		}
		if fc.ID == selectedFuncCall {
			styleName += ".selected"
		} else if fc.isPinned(config) {
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
		log.Printf("lines[%d]: %+v", len(lines), line)
		lines = append(lines, line)
	}
	return lines
}

// frames2funcs converts logutil.FuncStatusID to logutil.FuncID.
func (v *GraphView) frames2funcs(frames []logutil.FuncStatusID) (funcs []logutil.FuncID) {
	for _, id := range frames {
		fs, err := v.Root.Api.FuncStatus(v.LogID, strconv.Itoa(int(id)))
		if err != nil {
			log.Panic(err)
		}
		funcs = append(funcs, fs.Func)
	}
	return
}

// withFuncIDsはFuncCallのIDsを
func (v *GraphView) withFuncIDs(in chan restapi.FuncCall, out chan funcCallWithFuncIDs) {
	for fc := range in {
		funcs := v.frames2funcs(fc.Frames)
		out <- funcCallWithFuncIDs{
			FuncCall: fc,
			funcs:    funcs,
		}
	}
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
