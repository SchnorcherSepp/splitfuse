package fuse

import (
	"os"
	"time"
	"sync"
	"fmt"
	"path/filepath"

	"github.com/SchnorcherSepp/splitfuse/core"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
)

const maxLastFhCache = 12

// SplitFile wird von der Open() Funktion zurück gegeben
// und stellt die Read() Funktion zur verfügung..
type SplitFile struct {
	debug       bool
	dbFile      core.SfFile
	chunkFolder string
	chunkKeys   [][]byte
	chunkNames  [][]byte
	lastFh [maxLastFhCache]struct {
		fh        *os.File
		chunkNr   int
		nextChOff int64
	}
	nextFhIndex int
	lastFhMux sync.Mutex
	nodefs.File
}

// Release wird aufgerufen, wenn .close() auf die Datei im FUSE aufgerufen wird.
// Damit müssen auch alle offenen internen FH geschlossen werden.
// ACHTUNG: Muss syncronisiert werden!
func (f *SplitFile) Release() {
	f.lastFhMux.Lock() // THREAD SAFE: start
	for i := 0; i < maxLastFhCache; i++ {
		if f.lastFh[i].fh != nil {
			debug(f.debug, fmt.Sprintf("Release: close fh[%d] for %s", i, f.lastFh[i].fh.Name()))
			f.lastFh[i].fh.Close()
			f.lastFh[i].fh = nil
		}
	}
	f.lastFhMux.Unlock() // THREAD SAFE: end
}

// Read liest bytes und gibt sie fürs FUSE zurück.
// ACHTUNG: Muss syncronisiert werden!
func (f *SplitFile) Read(buf []byte, offset int64) (fuse.ReadResult, fuse.Status) {

	// leere Dateien sofort zurückgeben
	if f.dbFile.Size < 1 {
		debug(f.debug, "read empty file")
		return fuse.ReadResultData([]byte{}), fuse.OK
	}

	// Berechnungen
	readLength := int64(len(buf))
	chunkOffset := offset % core.CHUNKSIZE
	chunkNr := int((offset - chunkOffset) / core.CHUNKSIZE)

	// FIX: Es gibt den Fall, dass am Ende noch einmal 4096 bytes über die Datei gelesen werden.
	// Dabei kann es vorkommen, dass sich die ChunkNr erhöht und es dazu keine Daten in chunkKey und chunkName gibt.
	if chunkNr >= len(f.chunkKeys) {
		// würde panic: runtime error: index out of range auslösen
		debug(f.debug, "EOF FIX!")
		return fuse.ReadResultData([]byte{}), fuse.OK
	}

	// Daten ermitteln
	chunkKey := f.chunkKeys[chunkNr]
	chunkName := f.chunkNames[chunkNr]
	chunkNameHex := fmt.Sprintf("%x", chunkName)
	chunPath := filepath.Join(f.chunkFolder, chunkNameHex[:2], chunkNameHex)

	// Ich muss nun auf den chunk zugreifen und brauche dafür ein file-open
	// Da diese Operation teuer ist, speichere ich alte filehandler und verwende sie wieder, wenn es geht
	// Hier prüfe ich, ob es keinen Filehandler gibt  ODER  der erwartete offset nicht stimmt  ODER  ich auf einen anderen chunk zugreifen müsste
	f.lastFhMux.Lock() // THREAD SAFE: start

	foundPerfectFh := -1
	for i := 0; i < maxLastFhCache; i++ {
		if f.lastFh[i].fh != nil && f.lastFh[i].nextChOff == chunkOffset && f.lastFh[i].chunkNr == chunkNr {
			foundPerfectFh = i
			break
		}
	}

	if f.debug {
		debug(f.debug, fmt.Sprintf("use fh[%d] for position %d and len %d", foundPerfectFh, offset, len(buf)))
	}

	var openErr error
	if foundPerfectFh > -1 {
		// der gespeicherte file handler ist geeignet
		n, err := f.lastFh[foundPerfectFh].fh.Read(buf)
		if err != nil && n > 0 {
			debug(f.debug, fmt.Sprintf("ERROR: read error with recycled fh: n=%d, chunkOff=%d, chunkNr=%d, e=%s, p=%s", n, chunkOffset, chunkNr, err.Error(), chunPath))
			openErr = err
			f.lastFh[foundPerfectFh].fh = nil
		}
		// Buffer auf tatsächliche Länge kürzen
		buf = buf[:n]
		// next offset updaten
		f.lastFh[foundPerfectFh].nextChOff = chunkOffset + int64(len(buf))

	} else {
		// Plan B: neuer fh

		// alten fh schließen (wenn auf dem Platz einer wäre)
		if f.lastFh[f.nextFhIndex].fh != nil {
			if f.debug {
				currentPosition, _ := f.lastFh[f.nextFhIndex].fh.Seek(0, 1) // 0 offset to current position = current position
				debug(f.debug, fmt.Sprintf("close fh[%d] for %s at position %d", f.nextFhIndex, f.lastFh[f.nextFhIndex].fh.Name(), currentPosition))
			}
			f.lastFh[f.nextFhIndex].fh.Close()
			f.lastFh[f.nextFhIndex].fh = nil
		}

		// neuen fh öffnen, der auf Pos 0 kommt
		fh, err := os.Open(chunPath)
		f.lastFh[f.nextFhIndex].fh = fh
		if err != nil {
			debug(f.debug, fmt.Sprintf("ERROR: open error: chunkOff=%d, chunkNr=%d, e=%s, p=%s", chunkOffset, chunkNr, err.Error(), chunPath))
			openErr = err
			f.lastFh[f.nextFhIndex].fh = nil

		} else {
			// öffnen ok, weiter im Text
			debug(f.debug, fmt.Sprintf("open fh[%d] for %s", f.nextFhIndex, fh.Name()))

			// chunck offset setzen
			if _, err := fh.Seek(chunkOffset, 0); err != nil {
				debug(f.debug, fmt.Sprintf("ERROR: seek error: chunkOff=%d, chunkNr=%d, e=%s, p=%s", chunkOffset, chunkNr, err.Error(), chunPath))
				openErr = err
				f.lastFh[f.nextFhIndex].fh = nil

			} else {
				// phu, seek ist ok gegangen
				debug(f.debug, fmt.Sprintf("set fh[%d] offset to %d for %s", f.nextFhIndex, chunkOffset, fh.Name()))
				// Daten lesen
				n, err := fh.Read(buf)
				if err != nil && n > 0 {
					debug(f.debug, fmt.Sprintf("ERROR: read error with new fh: n=%d, chunkOff=%d, chunkNr=%d, e=%s, p=%s", n, chunkOffset, chunkNr, err.Error(), chunPath))
					openErr = err
					f.lastFh[f.nextFhIndex].fh = nil
				}
				// Buffer auf tatsächliche Länge kürzen
				buf = buf[:n]
			}
		}

		// fh speichern
		f.lastFh[f.nextFhIndex].chunkNr = chunkNr
		f.lastFh[f.nextFhIndex].nextChOff = chunkOffset + int64(len(buf))

		// f.nextFhIndex weiter setzen
		f.nextFhIndex++
		if f.nextFhIndex >= maxLastFhCache {
			f.nextFhIndex = 0
		}

	}
	f.lastFhMux.Unlock() // THREAD SAFE: end

	// gab es einen Fehler oben im Code, dann ist das schlecht
	if openErr != nil {
		// fehler zurückgeben
		debug(f.debug, "ERROR: "+openErr.Error())
		return fuse.ReadResultData([]byte{}), fuse.EIO
	}

	// die gelesenen Daten entschlüsseln
	core.CryptBytes(buf, chunkOffset, chunkKey)

	// SONDERFALL: was ist, wenn knapp über einen chunk hinaus gelesen werden soll?
	// dann muss eine weitere abfrage abgesetzt werden!
	nextChunkBufferSize := chunkOffset + readLength - core.CHUNKSIZE
	if nextChunkBufferSize > 0 {
		debug(f.debug, fmt.Sprintf("SPECIAL READ: %d", nextChunkBufferSize))

		// einen Puffer anlegen für meine eigenen Read() Funktion
		buf2 := make([]byte, nextChunkBufferSize)
		// ReadResult abholen
		res2, _ := f.Read(buf2, offset+readLength-nextChunkBufferSize)
		// []byte aus dem ReadResult extrahieren
		buf2, _ = res2.Bytes(buf2)
		// Göße des Puffers gegebenenfalls anpassen
		buf2 = buf2[:res2.Size()]

		// neuen großen Puffer anlegen
		buf = append(buf, buf2...)

		return fuse.ReadResultData(buf), fuse.OK
	}

	// NORMALFALL
	return fuse.ReadResultData(buf), fuse.OK
}

// SplitFs ist ein pathfs und hier sind fast alle eigenen FUSE Funktionen gebunden.
type SplitFs struct {
	debug        bool         // zusätzliche Meldungen einblenden
	db           core.SfDb    // Datenbank
	dbpath       string       // Pfad zur DB, um sie regelmäßig neu einzulesen
	intervall    int64        // update intervall in Sekunden  (bei 0 wird der Defaultwert genommen)
	lastDbUpdate int64        // wann wurde zuletzt checkDbUpdate() ausgeführt (Unix Time)
	lastDbMtime  int64        // die mtime des zuletzt geladenen DB files
	keyfile      core.KeyFile // Keyfile mit allen Schlüsseln
	chunkfolder  string       // Pfad zu den Chunks
	pathfs.FileSystem
}

// Diese Funktion wird von openDir getriggert
// Dabei stellt sie sicher, dass sie nur alle x sekunden einen Effekt hat
// return:
//   0 ... Erfolgreich
//   1 ... Intervall noch nicht erreicht
//   2 ... Fehler beim Lesen der mtime
//   3 ... DBfile existiert nicht
//   4 ... Fehler beim Laden der DB
func (fs *SplitFs) checkDbUpdate() int {
	// check intervall
	var intervall int64 = 5 * 60
	if fs.intervall > 0 {
		intervall = fs.intervall
	}

	// update nur alle 5 Minuten versuchen, egal ob erfolgreich oder nicht
	now := time.Now().Unix()
	thenPlus := fs.lastDbUpdate + intervall
	if thenPlus > now {
		// nur alle x Sekunden erlauben
		return 1
	}
	fs.lastDbUpdate = now

	// Funktionsaufruf melden (debug=true)
	debug(fs.debug, "check db update")

	// Hat sich die Datei verändert?
	// Nur aktualisierte Dateien laden
	info, err := os.Stat(fs.dbpath)
	if err != nil {
		// db file nicht da? ka. einfach abbrechen
		return 2
	}
	newDbMtime := info.ModTime().Unix()
	if newDbMtime == fs.lastDbMtime {
		// Datei ist noch gleich
		return 3
	}

	// Ladeversuch
	newdb, err := core.DbFromFile(fs.dbpath, fs.keyfile.DbKey())
	if err != nil {
		// db konnte nicht geladen werden
		// eventuell wird die Datei gerade erst geschrieben
		return 4
	}

	// neue DB setzen
	fs.db = newdb

	// ACHTUNG: Nachdem die DB gesetzt wurde, muss nun auch fs.lastDbMtime gespeichert werden
	// Vorher darf das nicht passieren, weil sonst die DB nicht geladen wird im Fehlerfall
	fs.lastDbMtime = newDbMtime

	// log schreiben (debug=true)
	debug(fs.debug, "update db")

	// bei Erfolg, true zurück geben
	return 0
}

// GetAttr gibt die File-Attribute fr Eintrge aus der DB zurück.
func (fs *SplitFs) GetAttr(name string, context *fuse.Context) (*fuse.Attr, fuse.Status) {
	// FIX: root
	if name == "" {
		name = "."
	}

	// Element in der DB suchen
	dbFile, ok := fs.db[name]
	if !ok {
		return nil, fuse.ENOENT
	}

	// Attribute setzen
	ret := &fuse.Attr{}

	// Basis-Attribute setzen
	ret.Size = dbFile.Size
	ret.Mtime = dbFile.Mtime
	ret.Ctime = dbFile.Mtime
	ret.Atime = dbFile.Mtime

	// Mode (Datei/Ordner)
	if dbFile.IsFile {
		ret.Mode = fuse.S_IFREG | 0644
		ret.Nlink = 1
	} else {
		ret.Mode = fuse.S_IFDIR | 0755
		ret.Nlink = uint32(len(dbFile.FolderContent))
	}

	return ret, fuse.OK
}

// OpenDir listet den Ordnerinhalt auf.
func (fs *SplitFs) OpenDir(name string, context *fuse.Context) (c []fuse.DirEntry, code fuse.Status) {
	// db update triggern
	fs.checkDbUpdate()

	// FIX: root
	if name == "" {
		name = "."
	}

	// Ordner in der DB suchen
	dbFile, ok := fs.db[name]
	if !ok {
		return nil, fuse.ENOENT
	}

	// prüfen, ob es e ein Ordner ist
	if dbFile.IsFile {
		return nil, fuse.ENOTDIR
	}

	// Enthaltene Elemente zurück geben
	l := len(dbFile.FolderContent)
	c = make([]fuse.DirEntry, 0, l)

	for _, v := range dbFile.FolderContent {
		// Sub-Element erzeugen
		tmp := fuse.DirEntry{Name: v.Name}
		// Mode setzen (Datei oder Ordner)
		// Nur das höchste Bit (eg. S_IFDIR) wird ausgewertet
		if v.IsFile {
			tmp.Mode = fuse.S_IFREG
		} else {
			tmp.Mode = fuse.S_IFDIR
		}
		// zu Liste hinzufügen
		c = append(c, tmp)
	}

	return c, fuse.OK
}

// Öffnet eine Datei und berechnet dabei alle Informationen, um auf die Chunks zuzugreifen.
func (fs *SplitFs) Open(name string, flags uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {

	// Datei in der DB suchen
	dbFile, ok := fs.db[name]
	if !ok {
		return nil, fuse.ENOENT
	}

	// prüfen, ob es e eine Datei ist
	if !dbFile.IsFile {
		return nil, fuse.ENOENT
	}

	// chunkkeys und chunknames berechnen
	chunkKeys := make([][]byte, len(dbFile.FileChunks))
	chunkNames := make([][]byte, len(dbFile.FileChunks))
	for i, chunkhash := range dbFile.FileChunks {
		chunkKeys[i] = fs.keyfile.CalcChunkKey(chunkhash[:])
		chunkNames[i] = fs.keyfile.CalcChunkCryptHash(chunkhash[:])
	}

	// Datei zurück geben
	return &SplitFile{
		File:        nodefs.NewDefaultFile(),
		debug:       fs.debug,
		chunkFolder: fs.chunkfolder,
		dbFile:      dbFile,
		chunkKeys:   chunkKeys,
		chunkNames:  chunkNames,
	}, fuse.OK
}

// Informationen für 'df -h'
func (fs *SplitFs) StatFs(name string) *fuse.StatfsOut {

	// Summe aller Dateien berechnen
	var sum uint64 = 0
	for _, v := range fs.db {
		sum += v.Size
	}

	// Dingige Dinge
	var blocksize uint64 = 8192
	var total uint64 = 109951162777600 // 100 TiB
	var free = total - sum

	return &fuse.StatfsOut{
		Blocks:  total / blocksize,
		Bfree:   free / blocksize,
		Bavail:  free / blocksize,
		Bsize:   uint32(blocksize),
		NameLen: 255,
		Frsize:  uint32(blocksize),
	}
}

// MountNormal greift auf Chunks zu und mountet die Klartextdateien
func MountNormal(dbpath string, keyfile string, chunkfolder string, mountpoint string, debug bool, test bool) *fuse.Server {

	// Prüft, ob der Chunk Ordner richtig ist
	// Es müssen die ganzen 00 .. ff Ordner vorhanden sein
	testfolder := []string{"00", "47", "83", "a0", "de", "ff"}
	for _, t := range testfolder {
		_, e := os.Stat(filepath.Join(chunkfolder, t))
		if e != nil {
			// Ordner existiert nicht
			panic("Wrong chunk folder! Can't find sub folder " + t)
		}
	}

	// Keyfile laden
	k := core.LoadKeyfile(keyfile)

	// DB laden
	db, err := core.DbFromFile(dbpath, k.DbKey())
	if err != nil {
		panic(err)
	}

	// OPTIONEN
	opts := &fuse.MountOptions{
		FsName:         "SplitFuse", // erste Spalte bei 'df -hT'
		Name:           "splitfsv2", // zweite Spalte bei 'df -hT'
		MaxReadAhead:   131072,
		Debug:          debug,
		AllowOther:     true,
		SingleThreaded: true,
	}

	// SplitFS erzeugen  (mit meinen Methoden)
	fs := &SplitFs{
		FileSystem:  pathfs.NewDefaultFileSystem(),
		debug:       debug,
		db:          db,
		dbpath:      dbpath,
		keyfile:     k,
		chunkfolder: chunkfolder,
	}

	// Als Zwischenschicht, (dann ist alles ein wenig einfacher), kommt NewPathNodeFs zum Einsatz
	nfs := pathfs.NewPathNodeFs(fs, nil)

	// NewFileSystemConnector erzeugen
	fsconn := nodefs.NewFileSystemConnector(nfs.Root(), nil)

	// FUSE mit den Optionen mounten
	server, err := fuse.NewServer(fsconn.RawFS(), mountpoint, opts)
	if err != nil {
		panic(err)
	}

	// loop (wartet auf EXIT)
	if !test {
		server.Serve()
	}

	return server
}
