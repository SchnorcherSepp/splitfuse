package fuse

import (
	"os"
	"fmt"
	"path/filepath"

	"github.com/SchnorcherSepp/splitfuse/core"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
)

// ReverseFile wird von der Open() Funktion zurück gegeben
// und stellt die Read() Funktion zur verfügung..
type ReverseFile struct {
	path     string
	chunkNr  int
	chunkKey []byte
	debug    bool
	nodefs.File
}

// ReverseFs ist ein pathfs und hier sind fast alle eigenen FUSE Funktionen gebunden.
type ReverseFs struct {
	crypHashIndex core.ReverseSfDb // um zu einem encChungHash einen Klartextpfad auflösen zu können
	rootdir       string           // Pfad zum rootdir
	db            core.SfDb        // Datenbank
	debug         bool
	pathfs.FileSystem
}

// Read liest bytes und gibt sie fürs FUSE zurück.
// ACHTUNG: Muss syncronisiert werden!
func (f *ReverseFile) Read(buf []byte, chunkOffset int64) (fuse.ReadResult, fuse.Status) {

	// file öffnen
	fh, err := os.Open(f.path)
	if err != nil {
		debug(f.debug, "can't open file: "+err.Error())
		return fuse.ReadResultData([]byte{}), fuse.EIO
	}
	defer fh.Close()

	// offset setzen
	offset := chunkOffset + int64(f.chunkNr)*core.CHUNKSIZE
	if _, err := fh.Seek(offset, 0); err != nil {
		debug(f.debug, "can't seek: "+err.Error())
		return fuse.ReadResultData([]byte{}), fuse.EIO
	}

	// Daten lesen
	n, err := fh.Read(buf)
	if err != nil && n > 0 {
		debug(f.debug, fmt.Sprintf("can't read file! p=%s, n=%d, e=%s", f.path, n, err.Error()))
		return fuse.ReadResultData([]byte{}), fuse.EIO
	}
	buf = buf[:n]

	// die gelesenen Daten verschlüsseln
	core.CryptBytes(buf, chunkOffset, f.chunkKey)

	// return
	return fuse.ReadResultData(buf), fuse.OK
}

// GetAttr gibt die File-Attribute für Einträge aus der DB zurück.
func (fs *ReverseFs) GetAttr(name string, context *fuse.Context) (*fuse.Attr, fuse.Status) {
	ret := &fuse.Attr{}
	ret.Mtime = 1490656554
	ret.Ctime = 1490656554
	ret.Atime = 1490656554

	// Daten aus dem crypHashIndex holen
	pai, err := fs.crypHashIndex.GetPAI(name)
	if err == nil {
		// Datei ist im PAI zu finden, also ist es eine Datei
		ret.Size = pai.ChunkSize
		ret.Mode = fuse.S_IFREG | 0644
	} else {
		// nicht im PAI, also ist es ein Ordner
		ret.Size = 4096
		ret.Mode = fuse.S_IFDIR | 0755
	}

	return ret, fuse.OK
}

// OpenDir listet den Ordnerinhalt auf.
func (fs *ReverseFs) OpenDir(name string, context *fuse.Context) (c []fuse.DirEntry, code fuse.Status) {

	// Ordner
	if len(name) == 0 {
		// eine Liste mit Strings: 00 bis ff  (alles klein)
		c := make([]fuse.DirEntry, 0, 256)
		for i := 0; i < 256; i++ {
			de := fuse.DirEntry{Name: fmt.Sprintf("%02x", i), Mode: fuse.S_IFDIR}
			c = append(c, de)
		}
		return c, fuse.OK
	}

	// Dateien
	if len(name) == 2 {
		for k := range fs.crypHashIndex {
			s := fmt.Sprintf("%x", k)
			de := fuse.DirEntry{Name: s, Mode: fuse.S_IFREG}
			if name[:2] == s[0:2] {
				c = append(c, de)
			}
		}
		return c, fuse.OK
	}

	// Sonstiges
	return c, fuse.ENOENT
}

// Öffnet eine Datei und berechnet dabei alle Informationen, um auf die Chunks zuzugreifen.
func (fs *ReverseFs) Open(name string, flags uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {

	// Daten aus dem crypHashIndex holen
	pai, err := fs.crypHashIndex.GetPAI(name)
	if err != nil {
		return nil, fuse.ENOENT
	}
	relpath := pai.Path
	chunkNr := pai.Index
	chunkKey := pai.ChunkKey

	// Hier prüfen wir, ob die Klartextdatei auf der Festplatte existiert
	// Hierzu erweitern wir den relpath zu einem Path.
	path := filepath.Join(fs.rootdir, relpath)
	if _, e := os.Stat(path); e != nil {
		// klartext Datei nicht auf der Festplatte
		return nil, fuse.ENOENT
	}

	// Datei zurück geben
	return &ReverseFile{
		File:     nodefs.NewDefaultFile(),
		path:     path,
		chunkNr:  chunkNr,
		chunkKey: chunkKey,
		debug:    fs.debug,
	}, fuse.OK
}

// Informationen für 'df -h'
func (fs *ReverseFs) StatFs(name string) *fuse.StatfsOut {

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

// MountReverse mountet die Chunks um sie in die CLoud zu syncronisieren
func MountReverse(dbpath string, keyfile string, rootdir string, mountdir string, debugFlag bool, test bool) *fuse.Server {

	// Keyfile laden
	k := core.LoadKeyfile(keyfile)

	// DB laden
	db, err := core.DbFromFile(dbpath, k.DbKey())
	if err != nil {
		panic(err)
	}

	// rootdir prüfen, indem ein Element in der DB gesucht wird
	for relpath := range db {
		// root Einträge überspringen
		if relpath == "" || relpath == "." {
			continue
		}
		// eine Prüfung machen
		path := filepath.Join(rootdir, relpath)
		if _, e := os.Stat(path); e != nil {
			panic("can't find element in rootdir: " + relpath)
		}
		// ende
		break
	}

	// crypHashIndex: Ich brauche eine Tabelle, in der ich den Chung Name (das ist der verschlüsselte Chunk Hash)
	// gesucht werden kann. Die DB kann das nicht leisten, also bauen wir uns eine neue Map.
	debug(debugFlag, "Optimize db for reverse mode. That can take a few minutes.")
	crypHashIndex := db.GetReverseSfDb(k)
	debug(debugFlag, "start mounting")

	// OPTIONEN
	opts := &fuse.MountOptions{
		FsName:     "ReverseFuse", // erste Spalte bei 'df -hT'
		Name:       "splitfsv2",   // zweite Spalte bei 'df -hT'
		Debug:      debugFlag,
		AllowOther: true,
	}

	// ReverseFS erzeugen  (mit meinen Methoden)
	fs := &ReverseFs{
		FileSystem:    pathfs.NewDefaultFileSystem(),
		crypHashIndex: crypHashIndex,
		rootdir:       rootdir,
		db:            db,
		debug:         debugFlag,
	}

	// Als Zwischenschicht, (dann ist alles ein wenig einfacher), kommt NewPathNodeFs zum Einsatz
	nfs := pathfs.NewPathNodeFs(fs, nil)

	// NewFileSystemConnector erzeugen
	fsconn := nodefs.NewFileSystemConnector(nfs.Root(), nil)

	// FUSE mit den Optionen mounten
	server, err := fuse.NewServer(fsconn.RawFS(), mountdir, opts)
	if err != nil {
		panic(err)
	}

	// loop (wartet auf EXIT)
	if !test {
		server.Serve()
	}
	return server
}

func debug(debug bool, msg string) {
	if debug {
		println("DEBUG: " + msg)
	}
}
