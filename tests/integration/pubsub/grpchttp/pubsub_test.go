package grpchttp

import (
	"context"
	"fmt"
	"os"
	"strings"
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
	"github.com/w-h-a/pkg/utils/httputils"
	"github.com/w-h-a/sidecar/tests/integration/pubsub/grpchttp/resources"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	servicePort int
	httpPort    int
	grpcPort    int

	httpSubscriber *resources.HttpSubscriber
)

func TestMain(m *testing.M) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		os.Exit(0)
	}

	var err error

	servicePort, err = runner.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	httpSubscriber = resources.NewHttpSubscriber(
		runner.ProcessWithId("http-subscriber"),
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
			"SERVICE_PROTOCOL": "http",
			"STORE":            "memory",
			"BROKER":           "memory",
			"CONSUMERS":        "go-a,go-b",
		}),
	)

	r := runner.NewTestRunner(
		runner.RunnerWithId("grpc-http pubsub"),
		runner.RunnerWithProcesses(
			httpSubscriber,
			sidecarProcess,
		),
	)

	os.Exit(r.Start(m))
}

func TestPubSubGrpctoHttp(t *testing.T) {
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
		rsp, err := httputils.HttpGet(fmt.Sprintf("127.0.0.1:%d/health/check", servicePort))
		if err != nil {
			return false
		}

		return string(rsp) == "ok"
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
					Data: &anypb.Any{
						Value: []byte(`{"status": "completed"}`),
					},
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
							Data: &anypb.Any{
								Value: []byte(fmt.Sprintf(`{"topic": "%s"}`, brokerName)),
							},
						},
					},
				),
			)

			pubRsp := &sidecar.PublishResponse{}

			err = grpcClient.Call(context.Background(), pubReq, pubRsp, client.CallWithAddress(fmt.Sprintf("127.0.0.1:%d", grpcPort)))
			require.NoError(c, err)

			event := httpSubscriber.Receive()

			data, ok := event.Event.Data.(map[string]interface{})
			require.True(t, ok)

			str, ok := data["topic"].(string)
			require.True(t, ok)

			rpl := strings.Replace(str, "-", "/", -1)

			require.True(c, fmt.Sprintf("/%s", rpl) == event.Route)

			require.True(c, event.Route == fmt.Sprintf("/%s", strings.Replace(event.Event.EventName, "-", "/", -1)))
		})
	}
}
