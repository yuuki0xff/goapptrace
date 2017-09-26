package srceditor

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func mkdir(elm ...string) {
	must(os.Mkdir(filepath.Join(elm...), os.ModePerm))
}

func touch(elm ...string) {
	must(ioutil.WriteFile(filepath.Join(elm...), []byte{}, os.ModePerm))
}

func mustInclude(t *testing.T, files []string, item ...string) {
	path := filepath.Join(item...)
	for i := range files {
		if files[i] == path {
			// ok
			return
		}
	}
	t.Errorf(fmt.Sprintf(`must include "%s" in %+v`, path, files))
}

func TestFindFiles(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "goapptrace-srceditor-test")
	if err != nil {
		panic(err)
	}

	// create test dir
	touch(tmpdir, "main.go")
	mkdir(tmpdir, "doc")
	touch(tmpdir, "doc", "index.html")
	mkdir(tmpdir, "testapp")
	touch(tmpdir, "testapp", "foo.go")
	mkdir(tmpdir, "testapp", "pkg0")
	mkdir(tmpdir, "testapp", "pkg1")
	touch(tmpdir, "testapp", "pkg1", "a.go")
	touch(tmpdir, "testapp", "pkg1", "b.go")
	touch(tmpdir, "testapp", "pkg1", "c.go.txt")
	touch(tmpdir, "testapp", "pkg1", "README.md")

	files, err := FindFiles(tmpdir)
	mustInclude(t, files, tmpdir, "main.go")
	mustInclude(t, files, tmpdir, "testapp", "foo.go")
	mustInclude(t, files, tmpdir, "testapp", "pkg1", "a.go")
	mustInclude(t, files, tmpdir, "testapp", "pkg1", "b.go")
	if len(files) != 4 {
		t.Errorf("expect len(files)=4, but %d ... %+v", len(files), files)
	}

	files, err = FindFiles(filepath.Join(tmpdir, "testapp", "foo.go"))
	must(err)
	mustInclude(t, files, tmpdir, "testapp", "foo.go")
	if len(files) != 1 {
		t.Errorf("expect len(files)=1, but %d ... %+v", len(files), files)
	}
}
