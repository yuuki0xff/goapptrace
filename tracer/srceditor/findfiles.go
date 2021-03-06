package srceditor

import (
	"os"
	"path/filepath"
	"strings"
)

const DefaultCaps = 1024

// returns all file paths of golang source in all sub-directories.
func FindFiles(fileOrDir string) ([]string, error) {
	files := make([]string, 0, DefaultCaps)

	stat, err := os.Stat(fileOrDir)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		if err := filepath.Walk(fileOrDir, func(fpath string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if strings.HasSuffix(fpath, ".go") {
				files = append(files, fpath)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	} else {
		files = []string{fileOrDir}
	}
	return files, nil
}
