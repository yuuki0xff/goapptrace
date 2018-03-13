package schema

import (
	"time"
)

// Logオブジェクトをmarshalするときに使用する。
// Logとは異なる点は、APIのレスポンスに必要なフィールドしか持っていないこと、および
// フィールドの値が更新されないため、ロックセずにフィールドの値にアクセスできることである。
// APIのレスポンスとして使用することを想定している。
type LogInfo struct {
	ID          string      `json:"log-id"`
	Version     int         `json:"version"`
	Metadata    LogMetadata `json:"metadata"`
	MaxFileSize int64       `json:"max-file-size"`
	ReadOnly    bool        `json:"read-only"`
}

type LogMetadata struct {
	// Timestamp of the last record
	Timestamp time.Time `json:"timestamp"`

	// The configuration of user interface
	UI UIConfig `json:"ui"`
}

type UIConfig struct {
	FuncCalls  map[FuncLogID]UIItemConfig `json:"func-calls"`
	Funcs      map[string]UIItemConfig    `json:"funcs"`
	Goroutines map[GID]UIItemConfig       `json:"goroutines"`
}

func (c *UIConfig) IsMasked(fc FuncLog) (masked bool) {
	var funcNames []string
	// TODO: fc.FramesをfuncNamesに変換

	for _, name := range funcNames {
		if f, ok := c.Funcs[name]; ok {
			masked = masked || f.Masked
		}
	}
	if g, ok := c.Goroutines[fc.GID]; ok {
		masked = masked || g.Masked
	}
	return
}
func (c *UIConfig) IsPinned(fc FuncLog) (pinned bool) {
	var funcNames []string
	// TODO: fc.FramesをfuncNamesに変換

	for _, name := range funcNames {
		if f, ok := c.Funcs[name]; ok {
			pinned = pinned || f.Pinned
		}
	}
	if g, ok := c.Goroutines[fc.GID]; ok {
		pinned = pinned || g.Pinned
	}
	return
}

type UIItemConfig struct {
	Pinned  bool   `json:"pinned"`
	Masked  bool   `json:"masked"`
	Comment string `json:"comment"`
}
