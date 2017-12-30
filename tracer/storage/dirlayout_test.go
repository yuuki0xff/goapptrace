package storage

import "testing"
import "github.com/go-playground/assert"

var (
	dr = DirLayout{
		Root: "/tmp/.goapptrace/logs",
	}

	goodStrID = "000102030405060708090a0b0c0d0e0f"
	goodFname = "000102030405060708090a0b0c0d0e0f.meta.json.gz"
	goodLogID = LogID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xA, 0xB, 0xC, 0xD, 0xE, 0xF}
)

func TestDirLayout_InfoFile(t *testing.T) {
	assert.Equal(t, File("/tmp/.goapptrace/logs/info.json.gz"), dr.InfoFile())
}

func TestDirLayout_MetaDir(t *testing.T) {
	assert.Equal(t, "/tmp/.goapptrace/logs/meta", dr.MetaDir())
}

func TestDirLayout_DataDir(t *testing.T) {
	assert.Equal(t, "/tmp/.goapptrace/logs/data", dr.DataDir())
}

func TestDirLayout_MetaID(t *testing.T) {
	badFnames := []string{
		"000102030405060708090a0b0c0d0e0f10.meta.json.gz",      // Too long ID
		"000102030405060708090a0b0c0d0e.meta.json.gz",          // Too short ID
		"000102030405060708090a0b0c0d0e0f.txt",                 // Wrong suffix
		"000102030405060708090a0b0c0d0e0f",                     // Has not suffix
		"prefix-000102030405060708090a0b0c0d0e0f.meta.json.gz", // Has prefix
		"INVALID.meta.json.gz",                                 // Invalid hex value
	}

	id, ok := dr.Fname2LogID(goodFname)
	assert.Equal(t, ok, true)
	assert.Equal(t, id, goodLogID)

	for _, badID := range badFnames {
		_, ok = dr.Fname2LogID(badID)
		if ok {
			t.Errorf("ID=%s: must be fail. but succeeded.", badID)
		}
	}
}

func TestDirLayout_MetaFile(t *testing.T) {
	assert.Equal(t, File("/tmp/.goapptrace/logs/meta/"+goodFname), dr.MetaFile(goodLogID))
}

func TestDirLayout_RawFuncLogFile(t *testing.T) {
	assert.Equal(t, File("/tmp/.goapptrace/logs/data/"+goodStrID+".0.rawfunc.log.gz"), dr.RawFuncLogFile(goodLogID, 0))
	assert.Equal(t, File("/tmp/.goapptrace/logs/data/"+goodStrID+".10.rawfunc.log.gz"), dr.RawFuncLogFile(goodLogID, 10))
}

func TestDirLayout_FuncLogFile(t *testing.T) {
	assert.Equal(t, File("/tmp/.goapptrace/logs/data/"+goodStrID+".0.func.log.gz"), dr.FuncLogFile(goodLogID, 0))
	assert.Equal(t, File("/tmp/.goapptrace/logs/data/"+goodStrID+".10.func.log.gz"), dr.FuncLogFile(goodLogID, 10))
}

func TestDirLayout_SymbolFile(t *testing.T) {
	assert.Equal(t, File("/tmp/.goapptrace/logs/data/"+goodStrID+".symbol.gz"), dr.SymbolFile(goodLogID))
}

func TestDirLayout_IndexFile(t *testing.T) {
	assert.Equal(t, File("/tmp/.goapptrace/logs/data/"+goodStrID+".index.gz"), dr.IndexFile(goodLogID))
}
