package fat

import (
	"errors"
	"io"
	"os"

	"github.com/diskfs/go-diskfs/filesystem/fat32"
	"github.com/diskfs/go-diskfs/util"
)

type FATReader interface {
	io.ReaderAt
	io.Seeker
}

func FileContentSection(r FATReader, site, blocksize int64, path string) (int64, int64, error) {
	var fsFile util.File
	fsFile = &nopWriter{r}
	fs, err := fat32.Read(fsFile, site, 0, blocksize)
	if err != nil {
		return 0, 0, err
	}
	file, err := fs.OpenFile(path, os.O_RDONLY)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()
	fat32File := file.(*fat32.File)

	return fat32File.GetContentSection()
}

type nopWriter struct {
	FATReader
}

func (w *nopWriter) WriteAt(p []byte, off int64) (n int, err error) {
	return 0, errors.New("reader is read-only")
}
