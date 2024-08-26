package controllers

import (
	pb "github.com/w-h-a/pkg/proto/action"
	"github.com/w-h-a/pkg/sidecar"
)

func DeserializeRecords(pairs []*pb.KeyVal) []sidecar.Record {
	records := []sidecar.Record{}

	for _, pair := range pairs {
		record := sidecar.Record{
			Key:   pair.Key,
			Value: pair.Value.Value,
		}
		records = append(records, record)
	}

	return records
}
