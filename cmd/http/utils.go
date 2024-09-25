package http

import (
	"encoding/json"

	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/store"
)

func SerializeRecords(recs []*store.Record) ([]sidecar.Record, error) {
	sidecarRecords := []sidecar.Record{}

	for _, record := range recs {
		sidecar := sidecar.Record{
			Key: record.Key,
		}

		var value interface{}

		if err := json.Unmarshal(record.Value, &value); err != nil {
			return nil, err
		}

		sidecar.Value = value

		sidecarRecords = append(sidecarRecords, sidecar)
	}

	return sidecarRecords, nil
}
