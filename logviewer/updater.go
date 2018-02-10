package logviewer

import (
	"context"
	"time"
)

const (
	DefaultUpdateInterval = 200 * time.Millisecond
)

type Updatable interface {
	// Update()を呼び出す頻度。
	// 負の値を返した場合、Update()は初回のみ呼び出され、それ以降の自動更新は行わない。
	// 0を返した場合、DefaultUpdateIntervalの頻度でUpdate()を呼び出す。
	// 正の値を返した場合、指定した頻度でUpdate()を呼び出す。
	UpdateInterval() time.Duration
	// Update()は、キャッシュのアップデートを行う。
	// 完了するまで長時間ブロックする可能性がある。
	// ctxはnon-nilでなければならない。
	Update(ctx context.Context)
}

type Updater struct{}

func (u Updater) Run(ctx context.Context, target Updatable) {
	d := target.UpdateInterval()
	if d == 0 {
		d = DefaultUpdateInterval
	}
	t := time.NewTimer(d)
	for {
		select {
		case <-t.C:
			target.Update(ctx)
		case <-ctx.Done():
			return
		}
	}
}
