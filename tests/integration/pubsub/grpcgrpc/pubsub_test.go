package grpcgrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/w-h-a/pkg/client"
	"github.com/w-h-a/pkg/client/grpcclient"
	"github.com/w-h-a/pkg/proto/health"
	"github.com/w-h-a/pkg/proto/sidecar"
	"github.com/w-h-a/pkg/runner"
	"github.com/w-h-a/pkg/runner/binary"
	"github.com/w-h-a/pkg/telemetry/log"
	"github.com/w-h-a/pkg/telemetry/log/memory"
	"github.com/w-h-a/pkg/utils/memoryutils"
	"github.com/w-h-a/sidecar/tests/integration/pubsub/grpcgrpc/resources"
)

var (
	servicePort int
	httpPort    int
	grpcPort    int

	grpcSubscriber *resources.GrpcSubscriber
)

func TestMain(m *testing.M) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		os.Exit(0)
	}

	logger := memory.NewLog(
		log.LogWithPrefix("integration test pubsub-grpc-grpc"),
		memory.LogWithBuffer(memoryutils.NewBuffer()),
	)

	log.SetLogger(logger)

	var err error

	servicePort, err = runner.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	grpcSubscriber = resources.NewGrpcSubscriber(
		runner.ProcessWithId("grpc-subscriber"),
		runner.ProcessWithEnvVars(map[string]string{
			"PORT": fmt.Sprintf("%d", servicePort),
		}),
	)

	httpPort, err = runner.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	grpcPort, err = runner.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	sidecarProcess := binary.NewProcess(
		runner.ProcessWithId("sidecar"),
		runner.ProcessWithUpBinPath("sidecar"),
		runner.ProcessWithUpArgs("sidecar"),
		runner.ProcessWithEnvVars(map[string]string{
			"NAMESPACE":        "default",
			"NAME":             "sidecar",
			"VERSION":          "v0.1.0-alpha.0",
			"HTTP_ADDRESS":     fmt.Sprintf(":%d", httpPort),
			"GRPC_ADDRESS":     fmt.Sprintf(":%d", grpcPort),
			"SERVICE_NAME":     "localhost",
			"SERVICE_PORT":     fmt.Sprintf("%d", servicePort),
			"SERVICE_PROTOCOL": "grpc",
			"STORE":            "memory",
			"BROKER":           "memory",
			"CONSUMERS":        "go-a,go-b",
			"SECRET":           "env",
		}),
	)

	r := runner.NewTestRunner(
		runner.RunnerWithId("grpc-grpc pubsub"),
		runner.RunnerWithProcesses(
			grpcSubscriber,
			sidecarProcess,
		),
	)

	os.Exit(r.Start(m))
}

func TestPubSubGrpcToGrpc(t *testing.T) {
	var err error

	grpcClient := grpcclient.NewClient()

	require.Eventually(t, func() bool {
		req := grpcClient.NewRequest(
			client.RequestWithNamespace("default"),
			client.RequestWithName("sidecar"),
			client.RequestWithMethod("Health.Check"),
			client.RequestWithUnmarshaledRequest(
				&health.HealthRequest{},
			),
		)

		rsp := &health.HealthResponse{}

		if err := grpcClient.Call(context.Background(), req, rsp, client.CallWithAddress(fmt.Sprintf("127.0.0.1:%d", grpcPort))); err != nil {
			return false
		}

		return rsp.Status == "ok"
	}, 10*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		req := grpcClient.NewRequest(
			client.RequestWithNamespace("default"),
			client.RequestWithName("grpc-subscriber"),
			client.RequestWithMethod("Health.Check"),
			client.RequestWithUnmarshaledRequest(
				&health.HealthRequest{},
			),
		)

		rsp := &health.HealthResponse{}

		if err := grpcClient.Call(context.Background(), req, rsp, client.CallWithAddress(fmt.Sprintf("127.0.0.1:%d", servicePort))); err != nil {
			return false
		}

		return rsp.Status == "ok"
	}, 10*time.Second, 10*time.Millisecond)

	// TODO: more in parallel
	t.Log("bad requests")

	pubReq := grpcClient.NewRequest(
		client.RequestWithNamespace("default"),
		client.RequestWithName("sidecar"),
		client.RequestWithMethod("Publish.Publish"),
		client.RequestWithUnmarshaledRequest(
			&sidecar.PublishRequest{
				Event: &sidecar.Event{
					EventName: "go-c",
					Payload:   []byte(`{"status": "completed"}`),
				},
			},
		),
	)

	pubRsp := &sidecar.PublishResponse{}

	err = grpcClient.Call(context.Background(), pubReq, pubRsp, client.CallWithAddress(fmt.Sprintf("127.0.0.1:%d", grpcPort)))
	require.Error(t, err)

	pt := runner.NewParallelTest(t)

	for _, brokerName := range []string{"go-a", "go-b"} {
		pt.Add(func(c *assert.CollectT) {
			t.Logf("good request for broker %s", brokerName)

			pubReq := grpcClient.NewRequest(
				client.RequestWithNamespace("default"),
				client.RequestWithName("sidecar"),
				client.RequestWithMethod("Publish.Publish"),
				client.RequestWithUnmarshaledRequest(
					&sidecar.PublishRequest{
						Event: &sidecar.Event{
							EventName: brokerName,
							Payload:   []byte(fmt.Sprintf(`{"topic": "%s"}`, brokerName)),
						},
					},
				),
			)

			pubRsp := &sidecar.PublishResponse{}

			err = grpcClient.Call(context.Background(), pubReq, pubRsp, client.CallWithAddress(fmt.Sprintf("127.0.0.1:%d", grpcPort)))
			require.NoError(c, err)

			event := grpcSubscriber.Receive()

			data := map[string]interface{}{}

			_ = json.Unmarshal(event.Event.Payload, &data)

			require.True(c, data["topic"] == event.Method)

			require.True(c, event.Method == event.Event.EventName)
		})
	}
}
