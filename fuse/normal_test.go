package fuse

import (
	"testing"
	"io/ioutil"
	"time"
	"os"
	"github.com/SchnorcherSepp/splitfuse/core"
	"path/filepath"
)

// Prüft ob die CheckUpdate Funktion wie geplant funktioniert
func TestCheckDbUpdate(t *testing.T) {
	path := filepath.Join(os.TempDir(), "updatetestfile.dat")

	fs := SplitFs{}
	fs.intervall = 2
	fs.debug = false
	fs.dbpath = path
	fs.keyfile = core.KeyFile{}

	// Rückgabewerte
	//   0 ... Erfolgreich
	//   1 ... Intervall noch nicht erreicht
	//   2 ... Fehler beim Lesen der mtime
	//   3 ... DBfile existiert nicht
	//   4 ... Fehler beim Laden der DB

	ioutil.WriteFile(path, []byte("hihi"), 0600) // defekte DB datei schreiben

	if s := fs.checkDbUpdate(); s != 4 { // defekte db
		t.Errorf("update test failed #1: status is %d", s)
	}
	if s := fs.checkDbUpdate(); s != 1 { // zeit noch nicht um
		t.Errorf("update test failed #2: status is %d", s)
	}
	time.Sleep(2100 * 1000000) // 2100 ms

	core.DbToFile(path, fs.keyfile.DbKey(), core.SfDb{}) // korrekte DB schreiben

	if s := fs.checkDbUpdate(); s != 0 { // korrekt geladen
		t.Errorf("update test failed #3: status is %d", s)
	}
	if s := fs.checkDbUpdate(); s != 1 { // zeit noch nicht um
		t.Errorf("update test failed #4: status is %d", s)
	}
	time.Sleep(2100 * 1000000) // 2100 ms

	if s := fs.checkDbUpdate(); s != 3 { // db file unverändert
		t.Errorf("update test failed #5: status is %d", s)
	}
	if s := fs.checkDbUpdate(); s != 1 { // zeit noch nicht um
		t.Errorf("update test failed #6: status is %d", s)
	}
	time.Sleep(2100 * 1000000) // 2100 ms

	os.Remove(path)

	if s := fs.checkDbUpdate(); s != 2 { // Datei existiert nicht
		t.Errorf("update test failed #7: status is %d", s)
	}
	if s := fs.checkDbUpdate(); s != 1 { // zeit noch nicht um
		t.Errorf("update test failed #8: status is %d", s)
	}

}
