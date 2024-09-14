package state

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/w-h-a/pkg/runner"
	"github.com/w-h-a/pkg/runner/docker"
	"github.com/w-h-a/pkg/telemetry/log"
	"github.com/w-h-a/pkg/utils/httputils"
)

const (
	numHealthChecks  = 60
	externalURL      = "http://localhost:3002"
	manyEntriesCount = 6
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
			Path: fmt.Sprintf("%s/resources/docker-compose-db.yml", dir),
		},
		{
			Path: fmt.Sprintf("%s/resources/docker-compose.yml", dir),
		},
	}

	r := docker.NewTestRunner(
		runner.RunnerWithId("state"),
		runner.RunnerWithFiles(testFiles...),
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
	States []State `json:"states,omitempty"`
}

type State struct {
	Key   string `json:"key,omitempty"`
	Value *Value `json:"value,omitempty"`
}

type Value struct {
	Data string `json:"data,omitempty"`
}

type SimpleKeyValue struct {
	Key   interface{}
	Value interface{}
}

func TestState(t *testing.T) {
	_, err := httputils.HttpGetNTimes(externalURL, numHealthChecks)
	require.NoError(t, err)

	testCases := generateTestCases()

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

				t.Logf("want %d", len(step.Expected.States))
				t.Logf("want %#+v", step.Expected.States)

				t.Logf("got %d", len(svcRsp.States))
				t.Logf("got %#+v", svcRsp.States)

				require.True(t, slicesEqual(step.Expected.States, svcRsp.States))

				time.Sleep(1 * time.Second)
			}
		})
	}
}

func generateTestCases() []Case {
	testCaseSingleKey := uuid.New().String()
	testCaseSingleValue := "The best song ever is 'Jockey Full of Bourbon' by Tom Waits"

	testCaseManyKeys := generateRandomStringKeys(manyEntriesCount)
	testCaseManyKeyValues := generateRandomStringValues(testCaseManyKeys)

	return []Case{
		{
			Name: "test empty create, list, and delete",
			Steps: []Step{
				{
					Command:  "create",
					Request:  RequestResponse{},
					Expected: RequestResponse{},
				},
				{
					Command:  "list",
					Request:  RequestResponse{},
					Expected: RequestResponse{},
				},
				{
					Command:  "delete",
					Request:  RequestResponse{},
					Expected: RequestResponse{},
				},
			},
		},
		{
			Name: "test single-item create, list, and delete",
			Steps: []Step{
				{
					Command:  "create",
					Request:  newRequest(SimpleKeyValue{testCaseSingleKey, testCaseSingleValue}),
					Expected: RequestResponse{},
				},
				{
					Command:  "list",
					Request:  RequestResponse{},
					Expected: newResponse(SimpleKeyValue{testCaseSingleKey, testCaseSingleValue}),
				},
				{
					Command:  "delete",
					Request:  newRequest(SimpleKeyValue{testCaseSingleKey, nil}),
					Expected: RequestResponse{},
				},
				{
					Command:  "list",
					Request:  RequestResponse{},
					Expected: RequestResponse{},
				},
			},
		},
		{
			Name: "test many-item create, list, and delete",
			Steps: []Step{
				{
					Command:  "create",
					Request:  newRequest(testCaseManyKeyValues...),
					Expected: RequestResponse{},
				},
				{
					Command:  "list",
					Request:  RequestResponse{},
					Expected: newResponse(testCaseManyKeyValues...),
				},
				{
					Command:  "delete",
					Request:  newRequest(testCaseManyKeys...),
					Expected: RequestResponse{},
				},
				{
					Command:  "list",
					Request:  RequestResponse{},
					Expected: RequestResponse{},
				},
			},
		},
	}
}

func newRequest(kvs ...SimpleKeyValue) RequestResponse {
	return newRequestResponse(kvs...)
}

func newResponse(kvs ...SimpleKeyValue) RequestResponse {
	return newRequestResponse(kvs...)
}

func newRequestResponse(kvs ...SimpleKeyValue) RequestResponse {
	states := make([]State, 0, len(kvs))

	for _, kv := range kvs {
		states = append(states, generateState(kv))
	}

	return RequestResponse{
		States: states,
	}
}

func generateState(kv SimpleKeyValue) State {
	if kv.Key == nil {
		return State{}
	}

	key := fmt.Sprintf("%v", kv.Key)

	if kv.Value == nil {
		return State{key, nil}
	}

	value := fmt.Sprintf("%v", kv.Value)

	return State{key, &Value{value}}
}

func generateRandomStringKeys(num int) []SimpleKeyValue {
	if num < 0 {
		return []SimpleKeyValue{}
	}

	output := make([]SimpleKeyValue, 0, num)

	for i := 1; i <= num; i++ {
		key := uuid.New().String()
		output = append(output, SimpleKeyValue{key, nil})
	}

	return output
}

func generateRandomStringValues(kvs []SimpleKeyValue) []SimpleKeyValue {
	output := make([]SimpleKeyValue, 0, len(kvs))

	for i, kv := range kvs {
		key := kv.Key
		value := fmt.Sprintf("value for entry #%d with key %v", i+1, key)
		output = append(output, SimpleKeyValue{key, value})
	}

	return output
}

func slicesEqual(want, got []State) bool {
	w := map[string]*Value{}

	g := map[string]*Value{}

	for _, pair := range want {
		w[pair.Key] = pair.Value
	}

	for _, pair := range got {
		g[pair.Key] = pair.Value
	}

	return reflect.DeepEqual(w, g)
}
