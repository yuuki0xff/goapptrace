package logviewer

import (
	"context"
	"log"
	"sync/atomic"
	"time"
)

// updateをUpdateInterval間隔で呼び出すworkerを起動する。
// workerの動作状況はrunning変数に保存される。
// 既に起動している場合は、何もしない。
func startAutoUpdateWorker(running *uint32, ctx context.Context, update func()) (started bool) {
	if !atomic.CompareAndSwapUint32(running, 0, 1) {
		// 既に起動済みなので何もしない
		return
	}
	started = true

	go func() {
		timer := time.NewTicker(UpdateInterval)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				update()
			case <-ctx.Done():
				break
			}
		}

		if !atomic.CompareAndSwapUint32(running, 1, 0) {
			log.Panic("invalid state")
		}
	}()
	return
}
