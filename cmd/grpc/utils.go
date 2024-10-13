package grpc

import (
	pb "github.com/w-h-a/pkg/proto/sidecar"
	pbTrace "github.com/w-h-a/pkg/proto/trace"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/store"
	"github.com/w-h-a/pkg/telemetry/tracev2"
	"google.golang.org/protobuf/types/known/anypb"
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

func SerializeRecords(recs []*store.Record) []*pb.KeyVal {
	pairs := []*pb.KeyVal{}

	for _, record := range recs {
		pair := &pb.KeyVal{
			Key:   record.Key,
			Value: &anypb.Any{Value: record.Value},
		}
		pairs = append(pairs, pair)
	}

	return pairs
}

func SerializeSecret(secret *sidecar.Secret) *pb.Secret {
	return &pb.Secret{
		Data: secret.Data,
	}
}

func SerializeSpan(s *tracev2.SpanData) *pbTrace.Span {
	return &pbTrace.Span{
		Name:     s.Name,
		Id:       s.Id,
		Parent:   s.Parent,
		Trace:    s.Trace,
		Started:  uint64(s.Started.UnixNano()),
		Ended:    uint64(s.Ended.UnixNano()),
		Metadata: s.Metadata,
	}
}
