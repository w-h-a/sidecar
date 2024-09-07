package helloworld

import (
	"encoding/json"
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
	if len(os.Getenv("E2E")) == 0 {
		os.Exit(0)
	}

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

type Case struct {
	Description string
	TestCommand string
	ExpectedRsp string
}

type AppResponse struct {
	Message   string `json:"message,omitempty"`
	StartTime int    `json:"start_time,omitempty"`
	EndTime   int    `json:"end_time,omitempty"`
}

type TestCommandRequest struct {
	Message string `json:"message,omitempty"`
}

func TestHelloWorld(t *testing.T) {
	helloTests := []Case{
		{
			Description: "when we call the blue endpoint",
			TestCommand: "blue",
			ExpectedRsp: "Hello, Blue",
		},
		{
			Description: "when we call the green endpoint",
			TestCommand: "green",
			ExpectedRsp: "Hello, Green",
		},
	}

	for _, testCase := range helloTests {
		t.Run(testCase.Description, func(t *testing.T) {
			_, err := httputils.HttpGet("http://localhost:3000")
			require.NoError(t, err)

			body, err := json.Marshal(TestCommandRequest{Message: "Hello!"})
			require.NoError(t, err)

			rsp, err := httputils.HttpPost(fmt.Sprintf("http://localhost:3000/tests/%s", testCase.TestCommand), body)
			require.NoError(t, err)

			var appRsp AppResponse

			err = json.Unmarshal(rsp, &appRsp)
			require.NoError(t, err)

			require.Equal(t, testCase.ExpectedRsp, appRsp.Message)
		})
	}
}
