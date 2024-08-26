package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/store"
	"github.com/w-h-a/pkg/utils/errorutils"
)

func BadRequest(w http.ResponseWriter, description string) {
	e := errorutils.BadRequest("action", description)
	w.WriteHeader(400)
	w.Write([]byte(e.Error()))
}

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
