package core

import (
	"os"
	"io"
	"bytes"
	"errors"
	"encoding/gob"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io/ioutil"
	"encoding/hex"
	"path/filepath"
)

// SfDb ist eine Map, dessen Key der Pfad eines Ordners oder einer Datei ist und
// dessen value ein SfFile Objekt ist. Das Root-Verzeichnis hat den Pfad: '.'
type SfDb map[string]SfFile

// SfFile enthält alle Daten, um eine Datei im FUSE darstellen und lesen zu können.
// Relevant sind nur die Attribute Size und Mtime, alles andere ist statisch.
// Ist das Objekt eine Datei, so wird FileChunks gesetzt. Ist es ein Ordner so ist FolderContent gesetzt.
type SfFile struct {
	// Attr
	Size  uint64 // size in bytes
	Mtime uint64 // time of last modification

	// file or folder
	IsFile        bool            // true is file, false is folder
	FileChunks    []ChunkHash     // if file: the full chunk list of this file
	FolderContent []FolderContent // if folder: a list ob sub elements of this folder
}

// ReverseSfDb bietet die Möglichkeit, zu einem verschlüsselten ChunkHash (das ist der Dateiname eines Chunks
// im ChunkStorage), den Pfad zur klartext Datei zu erfragen.
type ReverseSfDb map[ChunkHash]PathAndIndex

// PathAndIndex wird in der ReverseDB verwendet und speichert den Pfad zu einer Klartext Datei und
// welcher Chunk (index) davon gelesen werden muss. Dazu noch die Größe (getAttribut) und der ChunkKey.
type PathAndIndex struct {
	Path      string
	Index     int
	ChunkKey  []byte
	ChunkSize uint64
}

// ChunkHash ist ein sha512 Hash (64 bytes) über den Klartext eines Chunks.
// Eine Liste dieser Chunks ergeben eine ganze Datei.
type ChunkHash [64]byte

// FolderContent speichert den Namen eines Unter-Elements eines Ordners und
// ob es sich um eine Dateioder einen Ordner handelt.
type FolderContent struct {
	Name   string
	IsFile bool
}

// DbToEncGOB serialized und verschlüsselt das SfDb Objekt und gibt nonce und den ciphertext zurück.
// Im Fehlerfall wird ein Error zurück gegeben und der ciphertext ist Null.
func DbToEncGOB(key []byte, db SfDb) (nonce []byte, ciphertext []byte, err error) {

	// serialisiertes Objekt als bytes (plaintext)
	var plaintext = bytes.Buffer{}
	encoder := gob.NewEncoder(&plaintext)
	err = encoder.Encode(db)
	if err != nil {
		return
	}

	// create AES cipher with 16, 24, or 32 bytes key
	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}

	// Galois Counter Mode
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}

	// create random nonce with standard length
	nonce = make([]byte, aesgcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return
	}

	// encrypts and authenticates plaintext
	ciphertext = aesgcm.Seal(nil, nonce, plaintext.Bytes(), nil)

	// FIN
	return
}

// DbFromEncGOB entschlüsselt und authentisirt den ciphertext.
// Im Fehlerfall wird ein error zurück gegeben.
func DbFromEncGOB(key []byte, nonce []byte, ciphertext []byte) (db SfDb, err error) {

	// create AES cipher with 16, 24, or 32 bytes key
	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}

	// Galois Counter Mode
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}

	// decrypts and authenticates ciphertext
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return
	}

	// decode the plaintext and update db *SfDb
	decoder := gob.NewDecoder(bytes.NewReader(plaintext))
	err = decoder.Decode(&db)
	return
}

// DbToFile schreibt die DB in eine Datei.
// ACHTUNG: Das Ziel wird dabei überschrieben!
// Bei Problemen wird ein Fehler zurück gegeben der behandelt werden muss!
func DbToFile(path string, key []byte, db SfDb) error {
	// db verschlüsseln
	nonce, ciphertext, err := DbToEncGOB(key, db)
	if err != nil {
		return err
	}

	// Datei überschreiben
	fh, err := os.Create(path)
	defer fh.Close()
	if err != nil {
		return err
	}

	// nonce schreiben
	n, err := fh.Write(nonce)
	if err != nil {
		return err
	}
	if n != len(nonce) {
		return errors.New("write nonce failed")
	}

	// ciphertext schreiben
	n, err = fh.Write(ciphertext)
	if err != nil {
		return err
	}
	if n != len(ciphertext) {
		return errors.New("write ciphertext failed")
	}

	// FIN
	return nil
}

// DbFromFile liest eine Datei und gibt ein SfDB Objekt zurück.
// Im Fehlerfall wird ein Error zurck gegebe, der behandelt werden muss.
// Ein Beispiel für einen Fähler wäre, das Lesen einer noch nicht fertig geschriebenen DB Datei.
// Existiert die Datei überhaupt nicht, dann wird eine leere DB zurück gegeben
func DbFromFile(path string, key []byte) (db SfDb, err error) {
	gcmStandardNonceSize := 12

	// keine Datei -> leere DB
	_, err = os.Stat(path)
	if err != nil {
		// datei existiert nicht
		return SfDb{}, nil
	}

	// Datei öffnen
	fh, err := os.Open(path)
	defer fh.Close()
	if err != nil {
		return // z.B. error: file not found
	}

	// alles lesen
	filebytes, err := ioutil.ReadAll(fh)
	if err != nil {
		return // z.B. error: file to large
	}

	// Datei muss groß genug sein
	if len(filebytes) < gcmStandardNonceSize+1 {
		err = errors.New("db file is too short")
		return
	}

	// daten extrahieren
	nonce := filebytes[:gcmStandardNonceSize]
	ciphertext := filebytes[gcmStandardNonceSize:]

	// encrtypt
	db, err = DbFromEncGOB(key, nonce, ciphertext)
	if err != nil {
		return // z.B. error: Authentication failed
	}

	// FIN
	return
}

// Wandelt ein Sha512 Hash in ein [64]byte Array um.
func Sha512ToChunkHash(sha512 []byte) (ChunkHash, error) {
	// check input
	if len(sha512) != 64 {
		return ChunkHash{}, errors.New("sha512 hash must be 64 bytes long")
	}
	// build and return
	var retbytes [64]byte
	copy(retbytes[:], sha512)
	return retbytes, nil
}

// GetPAI erweitert ReverseSfDb und nimmt einen Pfad entgegen und sucht dann das richtige PAI Objekt
func (rdb *ReverseSfDb) GetPAI(name string) (PathAndIndex, error) {

	// Den ChunkName (entspricht dem verschlüsselten ChunkHash),
	// der als HEX-String vorliegt, muss in ein [64]byte umgewandelt werden
	h, err := hex.DecodeString(filepath.Base(name))
	if err != nil {
		// Umwandlung des HexString gescheitert
		return PathAndIndex{}, err
	}
	chunkname, err := Sha512ToChunkHash(h)
	if err != nil {
		return PathAndIndex{}, err
	}

	// Mit dem Chunknamen kann der relative Pfad der klartext Datei ermittelt werden
	// Die chunkNr beschreibt, welcher Teil der Klartextdatei gelesen werden muss
	pai, ok := (*rdb)[chunkname]
	if !ok {
		// in der DB nicht gefunden, also kann ich die Datei nicht öffnen
		return pai, errors.New("chunkname not found")
	}

	// return
	return pai, nil
}

// CalcChunkSize berechnet wie groß die einzelnen Chunks sind, bei einer bestimmten Klartextdateigröße
func CalcChunkSize(chunkNr int, fileSize uint64) (chunkSize uint64) {
	test1 := uint64(chunkNr+1) * CHUNKSIZE
	test2 := test1 - fileSize

	if test1 <= fileSize {
		return CHUNKSIZE
	}

	if test2 > CHUNKSIZE {
		return 0
	}

	return fileSize % CHUNKSIZE
}

// GetReverseSfDb erweitert SfDb und gibt eine ReverseSfDb der SfDb zurück.
func (db *SfDb) GetReverseSfDb(k KeyFile) ReverseSfDb {
	crypHashIndex := make(ReverseSfDb)

	// gehe alle klartext Dateien aus der DB durch
	for p, f := range *db {
		// Dateien, die 0 bytes haben, überspringen
		// auch Ordner überspringen
		if f.Size < 1 || !f.IsFile {
			continue
		}
		// und behandle von allen klartext Dateien
		// alle gespeicherten ChunkHashes  (Hash über den Klartext)
		for i, h := range f.FileChunks {
			// chunksize (ist er 0 bytes, dann nicht beachten)
			chunkSize := CalcChunkSize(i, f.Size)
			if chunkSize < 1 {
				continue
			}
			// dieser Hash muss verschlüsselt werden
			// und in ein core.ChunkHash konvertiert werden
			ch, _ := Sha512ToChunkHash(k.CalcChunkCryptHash(h[:]))
			// das ergibt dann den crypt Hash, der auch der Dateiname des Chunks ist
			crypHashIndex[ch] = PathAndIndex{Path: p, Index: i, ChunkKey: k.CalcChunkKey(h[:]), ChunkSize: chunkSize}
		}
	}

	return crypHashIndex
}
