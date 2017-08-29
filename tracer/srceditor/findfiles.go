package srceditor

import (
	"os"
	"path/filepath"
)

const DefaultCaps = 1024

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
			files = append(files, fpath)
			return nil
		}); err != nil {
			return nil, err
		}
	} else {
		files = []string{fileOrDir}
	}
	return files, nil
}
