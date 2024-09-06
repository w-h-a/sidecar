package helloworld

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/w-h-a/pkg/runner"
	"github.com/w-h-a/pkg/runner/docker"
	"github.com/w-h-a/pkg/telemetry/log"
	"github.com/w-h-a/pkg/utils/httputils"
)

func TestMain(m *testing.M) {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	testFiles := []runner.File{
		{
			Path: fmt.Sprintf("%s/resources/docker-compose.yml", dir),
		},
	}

	r := docker.NewTestRunner(
		runner.RunnerWithId("helloworld"),
		runner.RunnerWithFiles(testFiles...),
	)

	os.Exit(r.Start(m))
}

func TestHelloWorld(t *testing.T) {
	if len(os.Getenv("E2E")) == 0 {
		t.Skip("skipping e2e tests")
	}

	rsp, err := httputils.HttpGet("http://localhost:3000")
	require.NoError(t, err)

	require.Equal(t, rsp, []byte("Hello, World"))
}
