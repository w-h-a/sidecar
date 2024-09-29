package grpc

import (
	"context"
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
	"github.com/w-h-a/pkg/runner/http"
	"github.com/w-h-a/pkg/telemetry/log"
	"github.com/w-h-a/pkg/telemetry/log/memory"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	servicePort int
	httpPort    int
	grpcPort    int
)

func TestMain(m *testing.M) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		os.Exit(0)
	}

	logger := memory.NewLog(
		log.LogWithPrefix("integration test state-grpc"),
	)

	log.SetLogger(logger)

	var err error

	servicePort, err = runner.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	serviceProcess := http.NewProcess(
		runner.ProcessWithId("focal-service"),
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
			"DB":               "mydb",
			"STORES":           "mytable1,mytable2",
			"BROKER":           "memory",
			"SECRET":           "env",
		}),
	)

	r := runner.NewTestRunner(
		runner.RunnerWithId("state"),
		runner.RunnerWithProcesses(serviceProcess, sidecarProcess),
		runner.RunnerWithLogger(logger),
	)

	os.Exit(r.Start(m))
}

func TestStateGrpc(t *testing.T) {
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

	// TODO: more in parallel
	t.Log("bad requests")

	postReq := grpcClient.NewRequest(
		client.RequestWithNamespace("default"),
		client.RequestWithName("sidecar"),
		client.RequestWithMethod("State.Post"),
		client.RequestWithUnmarshaledRequest(
			&sidecar.PostStateRequest{
				Records: []*sidecar.KeyVal{
					{
						Key: "key1",
						Value: &anypb.Any{
							Value: []byte("value1"),
						},
					},
				},
			},
		),
	)

	postRsp := &sidecar.PostStateResponse{}

	err = grpcClient.Call(context.Background(), postReq, postRsp, client.CallWithAddress(fmt.Sprintf("127.0.0.1:%d", grpcPort)))
	require.Error(t, err)

	pt := runner.NewParallelTest(t)

	for _, storeName := range []string{"mytable1", "mytable2"} {
		pt.Add(func(c *assert.CollectT) {
			t.Logf("good request with store %s", storeName)

			postReq := grpcClient.NewRequest(
				client.RequestWithNamespace("default"),
				client.RequestWithName("sidecar"),
				client.RequestWithMethod("State.Post"),
				client.RequestWithUnmarshaledRequest(
					&sidecar.PostStateRequest{
						StoreId: storeName,
						Records: []*sidecar.KeyVal{
							{
								Key: "key1",
								Value: &anypb.Any{
									Value: []byte("value1"),
								},
							},
						},
					},
				),
			)

			postRsp := &sidecar.PostStateResponse{}

			err = grpcClient.Call(context.Background(), postReq, postRsp, client.CallWithAddress(fmt.Sprintf("127.0.0.1:%d", grpcPort)))
			require.NoError(c, err)

			getReq := grpcClient.NewRequest(
				client.RequestWithNamespace("default"),
				client.RequestWithName("sidecar"),
				client.RequestWithMethod("State.Get"),
				client.RequestWithUnmarshaledRequest(
					&sidecar.GetStateRequest{
						StoreId: storeName,
						Key:     "key1",
					},
				),
			)

			getRsp := &sidecar.GetStateResponse{}

			err = grpcClient.Call(context.Background(), getReq, getRsp, client.CallWithAddress(fmt.Sprintf("127.0.0.1:%d", grpcPort)))
			require.NoError(c, err)

			require.Equal(c, []byte("value1"), getRsp.Records[0].Value.Value)
		})
	}
}
