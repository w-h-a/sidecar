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
	"github.com/w-h-a/pkg/utils/errorutils"
	"google.golang.org/grpc/status"
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
		log.LogWithPrefix("integration test secret-grpc"),
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
			"SECRET_PREFIX":    "TEST_",
			"TEST_SECRET":      "mysecret",
		}),
	)

	r := runner.NewTestRunner(
		runner.RunnerWithId("state"),
		runner.RunnerWithProcesses(serviceProcess, sidecarProcess),
		runner.RunnerWithLogger(logger),
	)

	os.Exit(r.Start(m))
}

func TestSecretGrpc(t *testing.T) {
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

	pt := runner.NewParallelTest(t)

	for _, secretStoreName := range []string{"env", "fake"} {
		pt.Add(func(c *assert.CollectT) {
			t.Logf("request with secret store name %s", secretStoreName)

			getReq := grpcClient.NewRequest(
				client.RequestWithNamespace("default"),
				client.RequestWithName("sidecar"),
				client.RequestWithMethod("Secret.Get"),
				client.RequestWithUnmarshaledRequest(
					&sidecar.GetSecretRequest{
						SecretId: secretStoreName,
						Key:      "SECRET",
					},
				),
			)

			getRsp := &sidecar.GetSecretResponse{}

			err = grpcClient.Call(context.Background(), getReq, getRsp, client.CallWithAddress(fmt.Sprintf("127.0.0.1:%d", grpcPort)))

			if secretStoreName == "env" {
				require.NoError(c, err)

				require.Equal(c, "mysecret", getRsp.Secret.Data["SECRET"])
			} else {
				require.Error(c, err)

				e, ok := status.FromError(err)
				require.True(c, ok)

				internal := errorutils.ParseError(e.Message())
				require.Equal(c, int32(404), internal.Code)
				require.Equal(c, "Not Found", internal.Status)
				require.Equal(c, "component not found: fake", internal.Detail)
			}
		})
	}
}
