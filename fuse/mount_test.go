package fuse

import (
	"testing"
	"os"
	"math/rand"
	"crypto/sha512"
	"bytes"
	"encoding/hex"
	"path/filepath"
	"sort"
	"github.com/SchnorcherSepp/splitfuse/core"
	"github.com/SchnorcherSepp/splitfuse/copydir"
)

var (
	root   string
	disk   string
	mnt1   string
	mnt1cp string
	mnt2   string

	keyfilepath string
	dbfilepath  string

	validChunks [][]byte
	validFiles  [][]byte

	db  core.SfDb
	key core.KeyFile
)

func init() {
	keyfilepath = "../testdata/test.keyfile"

	// Pfade
	root = filepath.Join(os.TempDir(), "splitfuse-test")
	disk = filepath.Join(root, "diskfiles")
	mnt1 = filepath.Join(root, "mnt1")
	mnt1cp = filepath.Join(root, "mnt1-copy")
	mnt2 = filepath.Join(root, "mnt2")

	dbfilepath = filepath.Join(root, "test.dbfile")

	// Dateien schreiben
	writeBigTestFile("big1.testfile", core.CHUNKSIZE+26777216, "ed6690f9969f16ea03779ee94960f788d468a6e66f164ddb84a280b81b3abb069e2537a3c16c1d49fa98998d6461e36940318a60b2e479842701572cd1513314")
	writeBigTestFile("chun.testfile", core.CHUNKSIZE, "bc849819605df7daaf02cca25dc3e83f98ef94509f7cd6d4d5ae4e814a590a0c034d365a6e911316ed3d406e0fa21659de5caf8d6782a522f2309dd9215a8b52")
	writeBigTestFile("bufp.testfile", core.BUFFERSIZE+1, "18bf487f29220a4f231e00601900db58b3a474ce8467281f34d9c3739feabac2fcbb51e54f43fe7d3aa047bb87a19d9b6a506fc73ef5c1104a28d4fdba9e9c01")
	writeBigTestFile("buff.testfile", core.BUFFERSIZE, "9bbc5ddd0c84115b6f72da2c1d0812a98eae7a4da7ac348d77a5aecf40db4aad59b7b9d7366175e773ac7baa611f8272e0db2c1377735c162a7f3da61d0c6ad0")
	writeBigTestFile("smal.testfile", 17, "ba7c4e383bb76f180051195e23f4f45ee37c22184a502bb2ca868f28ccb1fac2b129b2033e57a5635538bbfd8da4ef0a8ef3f68be4aa51f5152209d15db86c59")
	writeBigTestFile("zero.testfile", 0, "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e")

	// Datenbank aufbaun
	key = core.LoadKeyfile(keyfilepath)
	var err error
	if db, err = core.DbFromFile(dbfilepath, key.DbKey()); err != nil {
		panic(err)
	}
	if db, _, _, err = core.ScanFolder(disk, db); err != nil {
		panic(err)
	}
	if err = core.DbToFile(dbfilepath, key.DbKey(), db); err != nil {
		panic(err)
	}

	// alle erlaubten hashes in eine liste
	var t []byte
	t, _ = hex.DecodeString("b7cb9135d73360dcd7c645ffa623d3f5f143b1c00a11f38c47aee38e69bd4c811f5e664dd8d0d0dbe58a9fd32eb61cf7f07c87c0f37923add2d6e7c3064ad3e2")
	validChunks = append(validChunks, t)
	t, _ = hex.DecodeString("f3afd46bc5ba9e16d8c7a843ee297e93c57b82070dced790b9c7375e709afe058218d32ce55d763da9c6c4603beca1e9f53ef3f7004155fe5530c6bc6a8772b1")
	validChunks = append(validChunks, t)
	t, _ = hex.DecodeString("e328c55e0073fea9de8938d9e0b03e5f15c90dc864a64b036f03cf1c08a75d84d3f472c1ab07da4919eda99e9a9bc11558c962e6680b84c92d3b8b70f4e985be")
	validChunks = append(validChunks, t)
	t, _ = hex.DecodeString("d69d5a530cfecdbc2fc4dc3960d0cd4775ba82768a2152bfb5ae466417de53994392b0a617b9d12ec79a3c96a584fa82584780819c7d3a63d3b92a1cfe9231e2")
	validChunks = append(validChunks, t)
	t, _ = hex.DecodeString("00a104f8bb40ce180c3db575df95d71f69d42e66813967cf1e7d7da19ca90b87ae6db0d535b3bfa4e4433d63b0b3ea836cbab9146f6304b6ab82e12d43b5fa53")
	validChunks = append(validChunks, t)

	// und auch für files
	t, _ = hex.DecodeString("ed6690f9969f16ea03779ee94960f788d468a6e66f164ddb84a280b81b3abb069e2537a3c16c1d49fa98998d6461e36940318a60b2e479842701572cd1513314")
	validFiles = append(validFiles, t)
	t, _ = hex.DecodeString("bc849819605df7daaf02cca25dc3e83f98ef94509f7cd6d4d5ae4e814a590a0c034d365a6e911316ed3d406e0fa21659de5caf8d6782a522f2309dd9215a8b52")
	validFiles = append(validFiles, t)
	t, _ = hex.DecodeString("18bf487f29220a4f231e00601900db58b3a474ce8467281f34d9c3739feabac2fcbb51e54f43fe7d3aa047bb87a19d9b6a506fc73ef5c1104a28d4fdba9e9c01")
	validFiles = append(validFiles, t)
	t, _ = hex.DecodeString("9bbc5ddd0c84115b6f72da2c1d0812a98eae7a4da7ac348d77a5aecf40db4aad59b7b9d7366175e773ac7baa611f8272e0db2c1377735c162a7f3da61d0c6ad0")
	validFiles = append(validFiles, t)
	t, _ = hex.DecodeString("ba7c4e383bb76f180051195e23f4f45ee37c22184a502bb2ca868f28ccb1fac2b129b2033e57a5635538bbfd8da4ef0a8ef3f68be4aa51f5152209d15db86c59")
	validFiles = append(validFiles, t)
	t, _ = hex.DecodeString("cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e")
	validFiles = append(validFiles, t)
}

// Der Readtest braucht eine große Datei, die zuerst angelegt werden muss.
func writeBigTestFile(name string, size int, hash string) string {

	os.Mkdir(root, 0777)
	os.Mkdir(disk, 0777)
	os.Mkdir(mnt1, 0777)
	os.Mkdir(mnt1cp, 0777)
	os.Mkdir(mnt2, 0777)
	path := filepath.Join(disk, name)

	// prüfen, ob die Datei existiert
	if _, e := os.Stat(path); e == nil {
		return path
	}

	// Datei neu anlegen
	hf := sha512.New()
	rand.Seed(1337)

	// Datei schreiben
	fh, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	for size > 0 {
		n := 1024 * 1024 * 2
		if n > size {
			n = size
		}
		buf := make([]byte, n)
		rand.Read(buf)
		fh.Write(buf)
		hf.Write(buf)
		size -= len(buf)

	}
	fh.Close()

	// Testfile prüfen
	hstr, _ := hex.DecodeString(hash)
	if !bytes.Equal(hf.Sum(nil), hstr) {
		os.Remove(path) // sonst würden falsche Dateien nicht mehr geschreiben
		panic("testfile hash is wrong: " + name)
	}

	return path
}

func findAllFiles(searchDir string) (folders []string, files []string) {
	// walk durchs verzeicnis
	err := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		// files und folder trennen
		if f.IsDir() {
			folders = append(folders, path)
		} else {
			files = append(files, path)
		}
		// weiter
		return nil
	})
	// error prüfen
	if err != nil {
		panic(err)
	}
	// sortieren
	sort.Strings(folders)
	sort.Strings(files)
	// return
	return
}

func fileHashCheck(t *testing.T, list []string, validHashList [][]byte) {
	// Dateihashes prüfen
	for _, p := range list {

		hf := sha512.New()

		fh, err := os.Open(p)
		if err != nil {
			t.Error(err)
		}

		for {
			// random buffer size
			bufsize := rand.Intn(17777777) + 1
			testbuf := make([]byte, bufsize)
			// read shit
			n, err := fh.Read(testbuf)
			if err != nil {
				break
			}
			// trimm buffer
			testbuf = testbuf[:n]
			// chalc hash
			hf.Write(testbuf)
		}
		fh.Close()

		// hash berechnen
		hash := hf.Sum(nil)

		// hashc in ok liste suchen
		ok := false
		for _, v := range validHashList {
			if bytes.Equal(hash, v) {
				ok = true
			}
		}

		// auswertung
		if !ok {
			t.Errorf("wrong hash:\n%s\n%x\n\n", p, hash)
		}
	}
}

// ================================================================================================================== //

func TestMountMux(t *testing.T) {
	/*
	 *  TODO: Diese Tests dauern extrem lange! Eventuell auskommentieren?
	 */
	mountReverse(t)
	mountNormal(t)
}

func mountReverse(t *testing.T) {

	// reverse mounten und daten einlesen
	server := MountReverse(dbfilepath, keyfilepath, disk, mnt1, false, true)
	go server.Serve()
	server.WaitMount()
	folders, files := findAllFiles(mnt1)

	// die zahl der folder muss immer gleich sein:
	// root (1) + lvl1 00-ff (256)
	if len(folders) != (1 + 256) {
		t.Errorf("wrong folder count: %d", len(folders))
	}

	// Zahl der dateien:
	// Es gibt 6 Testdateien. Davon wird eines auf zwei Dateien aufgeteilt (weil es zu groß ist), macht 7.
	// Aber big1 und chun teillen sich einen gemeinsamen chunk0. Daher bleiben es 6 Chunks die im chunk-storage sein müssen.
	// ABER: Eine Datei hat 0 bytes, und zu dieser dürfte es keinen Hash geben, also sind wir bei 5 Dateien
	if len(files) != 5 {
		t.Errorf("wrong files count: %d", len(files))
	}

	// Dateihashes prüfen
	go fileHashCheck(t, files, validChunks)
	fileHashCheck(t, files, validChunks)
	fileHashCheck(t, files, validChunks)

	// Chunks für später kopieren
	// und prüfen
	copydir.CopyDir(mnt1, mnt1cp)
	_, cpfiles := findAllFiles(mnt1cp)
	fileHashCheck(t, cpfiles, validChunks)

	// UNMOUNT
	server.Unmount()
}

func mountNormal(t *testing.T) {

	// mount NORMAL
	server := MountNormal(dbfilepath, keyfilepath, mnt1cp, mnt2, false, true)
	go server.Serve()
	server.WaitMount()

	// Verzeichnis auflisten
	folders, files := findAllFiles(mnt2)

	// die zahl der folder muss immer gleich sein:
	// root (1)
	if len(folders) != 1 {
		t.Errorf("wrong folder count: %d", len(folders))
	}

	// Zahl der dateien:
	// Es gibt 6 Testdateien.
	if len(files) != 6 {
		t.Errorf("wrong files count: %d", len(files))
	}

	// Dateihashes prüfen
	fileHashCheck(t, files, validFiles)

	// UMOUNT
	server.Unmount()
}
