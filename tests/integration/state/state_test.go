package state

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/w-h-a/pkg/client"
	"github.com/w-h-a/pkg/client/grpcclient"
	"github.com/w-h-a/pkg/proto/health"
	"github.com/w-h-a/pkg/proto/sidecar"
	"github.com/w-h-a/pkg/runner"
	"github.com/w-h-a/pkg/runner/binary"
	"github.com/w-h-a/pkg/runner/http"
	"github.com/w-h-a/pkg/telemetry/log"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	servicePort int
	httpPort    int
	rpcPort     int
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

	rpcPort, err = runner.GetFreePort()
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
			"RPC_ADDRESS":      fmt.Sprintf(":%d", rpcPort),
			"SERVICE_NAME":     "localhost",
			"SERVICE_PORT":     fmt.Sprintf("%d", servicePort),
			"SERVICE_PROTOCOL": "http",
			"STORE":            "memory",
			"DB":               "mydb",
			"STORES":           "mytable1,mytable2",
			"BROKER":           "memory",
		}),
	)

	r := runner.NewTestRunner(
		runner.RunnerWithId("state"),
		runner.RunnerWithProcesses(serviceProcess, sidecarProcess),
	)

	os.Exit(r.Start(m))
}

func TestStateRPC(t *testing.T) {
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

		if err := grpcClient.Call(context.Background(), req, rsp, client.CallWithAddress(fmt.Sprintf("127.0.0.1:%d", rpcPort))); err != nil {
			return false
		}

		return rsp.Status == "ok"
	}, 10*time.Second, 10*time.Millisecond)

	t.Run("good requests to mytable1", func(t *testing.T) {
		postReq := grpcClient.NewRequest(
			client.RequestWithNamespace("default"),
			client.RequestWithName("sidecar"),
			client.RequestWithMethod("State.Post"),
			client.RequestWithUnmarshaledRequest(
				&sidecar.PostStateRequest{
					StoreId: "mytable1",
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

		err = grpcClient.Call(context.Background(), postReq, postRsp, client.CallWithAddress(fmt.Sprintf("127.0.0.1:%d", rpcPort)))
		require.NoError(t, err)

		getReq := grpcClient.NewRequest(
			client.RequestWithNamespace("default"),
			client.RequestWithName("sidecar"),
			client.RequestWithMethod("State.Get"),
			client.RequestWithUnmarshaledRequest(
				&sidecar.GetStateRequest{
					StoreId: "mytable1",
					Key:     "key1",
				},
			),
		)

		getRsp := &sidecar.GetStateResponse{}

		err = grpcClient.Call(context.Background(), getReq, getRsp, client.CallWithAddress(fmt.Sprintf("127.0.0.1:%d", rpcPort)))
		require.NoError(t, err)

		require.Equal(t, []byte("value1"), getRsp.Records[0].Value.Value)
	})
}
