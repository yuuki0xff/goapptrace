package info

import (
	"os"
	"path/filepath"
)

const (
	AppName = "goapptrace"
	Version = "0.4.0-beta"

	DefaultStorageDir    = "~/goapptrace"
	DefaultLogfileEnv    = "GOAPPTRACE_LOG"
	DefaultLogfilePrefix = "./goapptrace"
	DefaultLogsrvEnv     = "GOAPPTRACE_SERVER"
	DefaultHttpDocRoot   = "./static/"
	DefaultExeName       = "exe"
	DefaultAppNameEnv    = "GOAPPTRACE_APP_NAME"
)

var (
	DocRootAbsPath, _ = filepath.Abs(filepath.Join(filepath.Dir(os.Args[0]), DefaultHttpDocRoot))
)
