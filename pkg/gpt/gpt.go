package gpt

import (
	"errors"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/partition"
	"github.com/diskfs/go-diskfs/partition/gpt"
)

func EFIPartitionSection(path string) (int64, int64, error) {
	disk, err := diskfs.Open(path, diskfs.WithOpenMode(diskfs.ReadOnly))
	if err != nil {
		return 0, 0, err
	}
	defer disk.File.Close()
	table, err := disk.GetPartitionTable()
	if err != nil {
		return 0, 0, err
	}
	part, err := findPartWithTypeGUID(table, gpt.EFISystemPartition)
	if err != nil {
		return 0, 0, err
	}
	return part.GetStart(), part.GetSize(), nil
}

func findPartWithTypeGUID(table partition.Table, guid gpt.Type) (*gpt.Partition, error) {
	for _, part := range table.GetPartitions() {
		part, ok := part.(*gpt.Partition)
		if !ok {
			return nil, errors.New("partition is not GPT")
		}
		if part.Type == guid {
			return part, nil
		}
	}
	return nil, errors.New("partition not found")
}
