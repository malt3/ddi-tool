package ddi

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/malt3/ddi-tool/pkg/cmdline"
	"github.com/malt3/ddi-tool/pkg/fat"
	"github.com/malt3/ddi-tool/pkg/gpt"
	"github.com/malt3/ddi-tool/pkg/uki"
)

type Image struct {
	path      string
	file      *os.File
	blocksize int64
	ukiPath   string
}

// New creates a new Image instance.
// imagePath is the path to the image file.
// blocksize is the blocksize of the image (usually 512, use 0 to enable autodetection).
// ukiPath is the path to the uki binary inside the EFI partition (usually /EFI/BOOT/BOOTX64.EFI).
func New(imagePath string, blocksize int64, ukiPath string) (*Image, error) {
	file, err := os.OpenFile(imagePath, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("opening image file: %w", err)
	}
	if blocksize == 0 {
		blocksize, err = learnBlocksize(file)
		if err != nil {
			return nil, fmt.Errorf("learning blocksize: %w", err)
		}
	}
	if ukiPath == "" {
		ukiPath = "/EFI/BOOT/BOOTX64.EFI"
	}
	return &Image{
		path:      imagePath,
		file:      file,
		blocksize: blocksize,
		ukiPath:   ukiPath,
	}, nil
}

func (i *Image) Close() error {
	return i.file.Close()
}

func (i *Image) GetCmdline() (*cmdline.Cmdline, error) {
	efiSectionStart, efiSectionSize, err := gpt.EFIPartitionSection(i.path)
	if err != nil {
		return nil, fmt.Errorf("finding cmdline: getting EFI partition section: %w", err)
	}

	efiPartition := io.NewSectionReader(i.file, efiSectionStart, efiSectionSize)

	fileContentOffset, fileContentSize, err := fat.FileContentSection(efiPartition, efiSectionStart, i.blocksize, i.ukiPath)
	if err != nil {
		return nil, fmt.Errorf("finding cmdline: getting file content section within EFI partition: %w", err)
	}

	ukiReader := io.NewSectionReader(i.file, efiSectionStart+fileContentOffset, fileContentSize)
	cmdlineOffset, cmdlineSize, err := uki.SectionBounds(ukiReader, ".cmdline")
	if err != nil {
		return nil, fmt.Errorf("finding cmdline: getting ,cmdline section within uki: %w", err)
	}

	return cmdline.New(
		cmdline.NewSectionHandle(i.file, efiSectionStart+fileContentOffset+cmdlineOffset, cmdlineSize),
		cmdlineSize,
	), nil
}

func learnBlocksize(r io.ReaderAt) (int64, error) {
	buf := make([]byte, 8)

	for bs := int64(512); bs <= 4096; bs *= 2 {
		_, err := r.ReadAt(buf, bs)
		if err != nil {
			return 0, err
		}
		if slices.Equal(buf, []byte("EFI PART")) {
			return bs, nil
		}
	}
	return 0, errors.New("blocksize not found")
}
