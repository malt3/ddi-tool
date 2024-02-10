package uki

import (
	"debug/pe"
	"errors"
	"io"
)

func SectionBounds(r io.ReaderAt, name string) (int64, int64, error) {
	file, err := pe.NewFile(r)
	if err != nil {
		return 0, 0, err
	}
	section := file.Section(name)
	if section == nil {
		return 0, 0, errors.New("section not found")
	}
	return int64(section.Offset), int64(section.VirtualSize), nil
}
