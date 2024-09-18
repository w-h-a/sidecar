package pubsub

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/w-h-a/pkg/runner"
	"github.com/w-h-a/pkg/runner/docker"
	"github.com/w-h-a/pkg/telemetry/log"
	"github.com/w-h-a/pkg/utils/httputils"
)

const (
	numHealthChecks           = 60
	publishURL                = "http://localhost:3003"
	subscribeURL              = "http://localhost:3004"
	topic                     = "arn:aws:sns:us-west-2:000000000000:dummy"
	randomOffsetMax           = 99
	numberOfMessagesToPublish = 6
	brokerWaitTime            = 60
	subscriberWaitTime        = 60
)

func TestMain(m *testing.M) {
	if len(os.Getenv("E2E")) == 0 {
		os.Exit(0)
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	brokerPath := fmt.Sprintf("%s/resources/docker-compose-broker.yml", dir)

	brokerProcess := docker.NewProcess(
		runner.ProcessWithUpBinPath("docker"),
		runner.ProcessWithUpArgs(
			"compose",
			"--file", brokerPath,
			"up",
			"--build",
			"--detach",
		),
		runner.ProcessWithDownBinPath("docker"),
		runner.ProcessWithDownArgs(
			"compose",
			"--file", brokerPath,
			"down",
			"--volumes",
		),
	)

	servicePath := fmt.Sprintf("%s/resources/docker-compose.yml", dir)

	serviceProcess := docker.NewProcess(
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

	r := docker.NewTestRunner(
		runner.RunnerWithId("pubsub"),
		runner.RunnerWithProcesses(brokerProcess, serviceProcess),
	)

	os.Exit(r.Start(m))
}

type PublishCommand struct {
	Topic string                 `json:"topic"`
	Data  map[string]interface{} `json:"data"`
}

type ReceivedMessagesResponse struct {
	ReceivedByTopicDummy []map[string]interface{} `json:"dummy-topic"`
}

type ServiceResponse struct {
	StartTime int    `json:"start_time,omitempty"`
	EndTime   int    `json:"end_time,omitempty"`
	Message   string `json:"message,omitempty"`
}

func TestPubSub(t *testing.T) {
	var err error

	_, err = httputils.HttpGetNTimes(publishURL, numHealthChecks)
	require.NoError(t, err)

	_, err = httputils.HttpGetNTimes(subscribeURL, numHealthChecks)
	require.NoError(t, err)

	t.Run("test pubsub", func(t *testing.T) {
		time.Sleep(brokerWaitTime * time.Second)

		sentMessages := sendToPubService(t)

		time.Sleep(subscriberWaitTime * time.Second)

		validateSubService(t, sentMessages)
	})
}

func sendToPubService(t *testing.T) ReceivedMessagesResponse {
	sentMessages := []map[string]interface{}{}

	commandBody := PublishCommand{Topic: topic}

	offset := rand.Intn(randomOffsetMax)

	url := fmt.Sprintf("%s/test", publishURL)

	for i := offset; i < offset+numberOfMessagesToPublish; i++ {
		commandBody.Data = map[string]interface{}{
			fmt.Sprintf("message %d", i): i,
		}

		sentMessages = append(sentMessages, commandBody.Data)

		bs, err := json.Marshal(commandBody)
		require.NoError(t, err)

		rsp, err := httputils.HttpPost(url, bs)
		require.NoError(t, err)

		var svcRsp ServiceResponse

		err = json.Unmarshal(rsp, &svcRsp)
		require.NoError(t, err)

		t.Logf("publish response %#+v", svcRsp)

		time.Sleep(1 * time.Second)
	}

	return ReceivedMessagesResponse{
		ReceivedByTopicDummy: sentMessages,
	}
}

func validateSubService(t *testing.T, sentMessages ReceivedMessagesResponse) {
	url := fmt.Sprintf("%s/test", subscribeURL)

	rsp, err := httputils.HttpPost(url, nil)
	require.NoError(t, err)

	var svcRsp ReceivedMessagesResponse

	err = json.Unmarshal(rsp, &svcRsp)
	require.NoError(t, err)

	t.Logf("want %d", len(sentMessages.ReceivedByTopicDummy))
	t.Logf("want %#+v", sentMessages.ReceivedByTopicDummy)

	t.Logf("got %d", len(svcRsp.ReceivedByTopicDummy))
	t.Logf("got %#+v", svcRsp.ReceivedByTopicDummy)

	require.True(t, slicesEqual(sentMessages.ReceivedByTopicDummy, svcRsp.ReceivedByTopicDummy))
}

func slicesEqual(want, got []map[string]interface{}) bool {
	w := map[string]float64{}

	g := map[string]float64{}

	for _, pair := range want {
		for k, v := range pair {
			w[k] = float64(v.(int))
		}
	}

	for _, pair := range got {
		for k, v := range pair {
			g[k] = v.(float64)
		}
	}

	return reflect.DeepEqual(w, g)
}
