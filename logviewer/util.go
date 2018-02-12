package logviewer

import (
	"context"
	"time"
)

// updateをUpdateInterval間隔で呼び出すworker。
// 最初にUpdate()を呼び出すのは、upd.UpdateInterval()経過後である。
// このメソッドはctxが完了するまで制御を返さない。
func updateWorker(ctx context.Context, upd Updatable) {
	interval := upd.UpdateInterval()
	if interval <= 0 {
		interval = DefaultUpdateInterval
	}
	timer := time.NewTicker(interval)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			upd.Update(ctx)
		case <-ctx.Done():
			return
		}
	}
	// unreachable
}
