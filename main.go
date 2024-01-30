package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/malt3/ddi-tool/fat"
	"github.com/malt3/ddi-tool/gpt"
	"github.com/malt3/ddi-tool/uki"
)

var (
	image     = flag.String("path", "image.raw", "path to the image")
	ukiPath   = flag.String("uki", "/EFI/BOOT/BOOTX64.EFI", "path to the uki binary")
	blocksize = flag.Int64("blocksize", 512, "blocksize of the image")
)

func main() {
	flag.Parse()

	efiSectionStart, efiSectionSize, err := gpt.EFIPartitionSection(*image)
	if err != nil {
		panic(err)
	}

	imageFile, err := os.Open(*image)
	if err != nil {
		panic(err)
	}
	defer imageFile.Close()

	efiPartition := io.NewSectionReader(imageFile, efiSectionStart, efiSectionSize)

	fileContentOffset, fileContentSize, err := fat.FileContentSection(efiPartition, efiSectionStart, *blocksize, *ukiPath)
	if err != nil {
		panic(err)
	}

	ukiReader := io.NewSectionReader(imageFile, efiSectionStart+fileContentOffset, fileContentSize)
	cmdlineOffset, cmdlineSize, err := uki.SectionBounds(ukiReader, ".cmdline")
	if err != nil {
		panic(err)
	}

	fmt.Printf("offset: %d, size: %d\n", efiSectionStart+fileContentOffset+cmdlineOffset, cmdlineSize)
	// TODO: remove unused contents slice
	contents := make([]byte, cmdlineSize)
	if _, err := imageFile.ReadAt(contents, efiSectionStart+fileContentOffset+cmdlineOffset); err != nil {
		panic(err)
	}
	fmt.Println(string(contents))
}
