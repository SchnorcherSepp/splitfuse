package main

import (
	"os"
	"path/filepath"

	"github.com/SchnorcherSepp/splitfuse/core"
	"github.com/SchnorcherSepp/splitfuse/fuse"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app   = kingpin.New(filepath.Base(os.Args[0]), "Ein Kommandozeilen-Tool zum Verwalten und Mounten von SplitFUSE.")
	debug = app.Flag("debug", "Aktiviert den Debug-Mode bei FUSE").Bool()

	gen        = app.Command("newkey", "Erstellt ein neues Keyfile für SplitFuse")
	genKeyfile = gen.Flag("keyfile", "Pfad zum Keyfile (Datei darf noch NICHT existieren)").Required().String()

	scan        = app.Command("scan", "Scant einen Ordner und aktualisiert gegebebenfalls die DB")
	scanDB      = scan.Flag("dbfile", "Pfad zur DB (wird überschrieben)").Required().String()
	scanKeyfile = scan.Flag("keyfile", "Pfad zum Keyfile").Required().ExistingFile()
	scanRoot    = scan.Flag("rootdir", "Pfad zum Root-Ordner mit allen Klartext Dateien").Required().ExistingDir()

	normal       = app.Command("normal", "Mountet Klartext Dateien")
	normalDB     = normal.Flag("dbfile", "Pfad zur DB. Die Datei wird regelmäßig neu eingelesen.").Required().ExistingFile()
	normalKey    = normal.Flag("keyfile", "Pfad zum Keyfile").Required().ExistingFile()
	normalChunks = normal.Flag("chunkdir", "Pfad zum Ordner mit allen notwendigen Chunks (eventuell CloudMount)").Required().ExistingDir()
	normalMount  = normal.Flag("mountdir", "Ordner, in dem die Klartext Dateien gemountet werden sollen").Required().ExistingDir()

	reverse      = app.Command("reverse", "Mountet den Chunk-Ordner um die Chunks mit der Cloud syncronisieren zu können")
	reverseDB    = reverse.Flag("dbfile", "Pfad zur DB").Required().ExistingFile()
	reverseKey   = reverse.Flag("keyfile", "Pfad zum Keyfile").Required().ExistingFile()
	reverseRoot  = reverse.Flag("rootdir", "Pfad zum Root-Ordner mit allen Klartext Dateien").Required().ExistingDir()
	reverseMount = reverse.Flag("mountdir", "Ordner, in dem die Chunks gemountet werden sollen").Required().ExistingDir()
)

func main() {
	app.Version("splitfuse 2.2.1")
	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	switch command {
	case gen.FullCommand():
		// neues keyfile schreiben
		core.NewRandomKeyfile(*genKeyfile)

	case scan.FullCommand():
		// keyfile laden
		k := core.LoadKeyfile(*scanKeyfile)
		// alte DB laden
		oldDB, err := core.DbFromFile(*scanDB, k.DbKey(), )
		if err != nil {
			panic(err)
		}
		// ordern scannen
		newDB, changed, summary, err := core.ScanFolder(*scanRoot, oldDB, *debug)
		if err != nil {
			panic(err)
		}
		// gibt es änderungen?
		if changed {
			print("update DB: ")
			println(summary)
			err = core.DbToFile(*scanDB, k.DbKey(), newDB)
			if err != nil {
				panic(err)
			}
		}

	case normal.FullCommand():
		fuse.MountNormal(*normalDB, *normalKey, *normalChunks, *normalMount, *debug, false)

	case reverse.FullCommand():
		fuse.MountReverse(*reverseDB, *reverseKey, *reverseRoot, *reverseMount, *debug, false)
	}

}
