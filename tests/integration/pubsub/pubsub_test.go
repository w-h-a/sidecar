package pubsub

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		os.Exit(0)
	}

	os.Exit(m.Run())
}
