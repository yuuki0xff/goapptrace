package info

import (
	"os"
	"path/filepath"
)

const (
	APP_NAME = "goapptrace"
	VERSION  = "0.0.1"

	DEFAULT_CONFIG_DIR     = "./.goapptrace"
	DEFAULT_LOGFILE_ENV    = "GOAPPTRACE_LOG"
	DEFAULT_LOGFILE_PREFIX = "./goapptrace"
	DEFAULT_LOGSRV_ENV     = "GOAPPTRACE_SERVER"
	DEFAULT_HTTP_DOC_ROOT  = "./static/"
	DEFAULT_EXE_NAME       = "exe"
	DEFAULT_APP_NAME_ENV   = "GOAPPTRACE_APP_NAME"
)

var (
	DocRootAbsPath, _ = filepath.Abs(filepath.Join(filepath.Dir(os.Args[0]), DEFAULT_HTTP_DOC_ROOT))
)
