package secret

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/w-h-a/pkg/runner"
	"github.com/w-h-a/pkg/runner/docker"
	"github.com/w-h-a/pkg/telemetry/log"
	"github.com/w-h-a/pkg/telemetry/log/memory"
	"github.com/w-h-a/pkg/utils/httputils"
	"github.com/w-h-a/pkg/utils/memoryutils"
)

const (
	numHealthChecks = 60
	externalURL     = "http://localhost:3005"
	secretWaitTime  = 60
)

func TestMain(m *testing.M) {
	if len(os.Getenv("E2E")) == 0 {
		os.Exit(0)
	}

	logger := memory.NewLog(
		log.LogWithPrefix("e2e test secret"),
		memory.LogWithBuffer(memoryutils.NewBuffer()),
	)

	log.SetLogger(logger)

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	secretPath := fmt.Sprintf("%s/resources/docker-compose-secret.yml", dir)

	secretProcess := docker.NewProcess(
		runner.ProcessWithId(secretPath),
		runner.ProcessWithUpBinPath("docker"),
		runner.ProcessWithUpArgs(
			"compose",
			"--file", secretPath,
			"up",
			"--build",
			"--detach",
		),
		runner.ProcessWithDownBinPath("docker"),
		runner.ProcessWithDownArgs(
			"compose",
			"--file", secretPath,
			"down",
			"--volumes",
		),
	)

	servicePath := fmt.Sprintf("%s/resources/docker-compose.yml", dir)

	serviceProcess := docker.NewProcess(
		runner.ProcessWithId(servicePath),
		runner.ProcessWithUpBinPath("docker"),
		runner.ProcessWithUpArgs(
			"compose",
			"--file", servicePath,
			"up",
			"--build",
			"--detach",
		),
		runner.ProcessWithDownBinPath("docker"),
		runner.ProcessWithDownArgs(
			"compose",
			"--file", servicePath,
			"down",
			"--volumes",
		),
	)

	r := runner.NewTestRunner(
		runner.RunnerWithId("secret"),
		runner.RunnerWithProcesses(secretProcess, serviceProcess),
	)

	os.Exit(r.Start(m))
}

type Case struct {
	Name  string
	Steps []Step
}

type Step struct {
	Command  string
	Request  RequestResponse
	Expected RequestResponse
}

type RequestResponse struct {
	Secrets []Secret `json:"secrets,omitempty"`
}

type Secret struct {
	Store string            `json:"store,omitempty"`
	Key   string            `json:"key,omitempty"`
	Data  map[string]string `json:"data,omitempty"`
}

type SimpleKeyValue struct {
	Key   interface{}
	Value interface{}
}

func TestSecret(t *testing.T) {
	_, err := httputils.HttpGetNTimes(externalURL, numHealthChecks)
	require.NoError(t, err)

	testCases := generateTestCases()

	time.Sleep(secretWaitTime * time.Second)

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			for _, step := range testCase.Steps {
				body, err := json.Marshal(step.Request)
				require.NoError(t, err)

				url := fmt.Sprintf("%s/test/%s", externalURL, step.Command)

				rsp, err := httputils.HttpPost(url, body)
				require.NoError(t, err)

				var svcRsp RequestResponse

				err = json.Unmarshal(rsp, &svcRsp)
				require.NoError(t, err)

				t.Logf("want %d", len(step.Expected.Secrets))
				t.Logf("want %#+v", step.Expected.Secrets)

				t.Logf("got %d", len(svcRsp.Secrets))
				t.Logf("got %#+v", svcRsp.Secrets)

				require.True(t, slicesEqual(step.Expected.Secrets, svcRsp.Secrets))
			}
		})
	}
}

func generateTestCases() []Case {
	return []Case{
		{
			Name: "test wrong secret store",
			Steps: []Step{
				{
					Command:  "get",
					Request:  newRequest("env", SimpleKeyValue{Key: "dummy"}),
					Expected: newResponse("env", SimpleKeyValue{Key: "dummy"}),
				},
			},
		},
		{
			Name: "test a secret",
			Steps: []Step{
				{
					Command:  "get",
					Request:  newRequest("ssm", SimpleKeyValue{Key: "dummy"}),
					Expected: newResponse("ssm", SimpleKeyValue{Key: "dummy", Value: map[string]string{"dummy": "secret"}}),
				},
			},
		},
	}
}

func newRequest(store string, kvs ...SimpleKeyValue) RequestResponse {
	return newRequestResponse(store, kvs...)
}

func newResponse(store string, kvs ...SimpleKeyValue) RequestResponse {
	return newRequestResponse(store, kvs...)
}

func newRequestResponse(store string, kvs ...SimpleKeyValue) RequestResponse {
	secrets := make([]Secret, 0, len(kvs))

	for _, kv := range kvs {
		secrets = append(secrets, generateSecret(store, kv))
	}

	return RequestResponse{
		Secrets: secrets,
	}
}

func generateSecret(store string, kv SimpleKeyValue) Secret {
	if kv.Key == nil {
		return Secret{}
	}

	key := fmt.Sprintf("%v", kv.Key)

	if kv.Value == nil {
		return Secret{store, key, nil}
	}

	value := kv.Value.(map[string]string)

	return Secret{store, key, value}
}

func slicesEqual(want, got []Secret) bool {
	w := map[string]map[string]string{}

	g := map[string]map[string]string{}

	for _, pair := range want {
		w[pair.Key] = pair.Data
	}

	for _, pair := range got {
		g[pair.Key] = pair.Data
	}

	return reflect.DeepEqual(w, g)
}
