package cmd

import (
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/urfave/cli"
	"github.com/w-h-a/pkg/api"
	"github.com/w-h-a/pkg/api/httpapi"
	"github.com/w-h-a/pkg/broker"
	"github.com/w-h-a/pkg/client/grpcclient"
	"github.com/w-h-a/pkg/client/httpclient"
	"github.com/w-h-a/pkg/server"
	"github.com/w-h-a/pkg/server/grpcserver"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/sidecar/custom"
	"github.com/w-h-a/pkg/store"
	"github.com/w-h-a/pkg/telemetry/log"
	"github.com/w-h-a/sidecar/cmd/config"
	"github.com/w-h-a/sidecar/cmd/http"
	"github.com/w-h-a/sidecar/cmd/rpc"
)

func run(ctx *cli.Context) {
	// get clients
	httpClient := httpclient.NewClient()

	grpcClient := grpcclient.NewClient()

	stores := map[string]store.Store{}

	brokers := map[string]broker.Broker{}

	st, err := GetStoreBuilder(config.Store)
	if err != nil {
		log.Fatal(err)
	}

	for _, s := range config.Stores {
		if len(s) == 0 {
			continue
		}

		stores[s] = MakeStore(st, []string{config.StoreAddress}, config.DB, s)
	}

	bk, err := GetBrokerBuilder(config.Broker)
	if err != nil {
		log.Fatal(err)
	}

	for _, s := range config.Producers {
		if len(s) == 0 {
			continue
		}

		brokers[s] = MakeProducer(bk, []string{config.BrokerAddress}, s)
	}

	for _, s := range config.Consumers {
		if len(s) == 0 {
			continue
		}

		brokers[s] = MakeConsumer(bk, []string{config.BrokerAddress}, s, config.Broker == "memory")
	}

	// get services
	_, httpPort, _ := strings.Cut(config.HttpAddress, ":")
	_, rpcPort, _ := strings.Cut(config.RpcAddress, ":")

	sidecarOpts := []sidecar.SidecarOption{
		sidecar.SidecarWithServiceName(config.ServiceName),
		sidecar.SidecarWithHttpPort(sidecar.Port{Port: httpPort}),
		sidecar.SidecarWithRpcPort(sidecar.Port{Port: rpcPort}),
		sidecar.SidecarWithServicePort(sidecar.Port{Port: config.ServicePort, Protocol: config.ServiceProtocol}),
		sidecar.SidecarWithStores(stores),
		sidecar.SidecarWithBrokers(brokers),
	}

	if config.ServiceProtocol == "rpc" {
		sidecarOpts = append(sidecarOpts, sidecar.SidecarWithClient(grpcClient))
	} else {
		sidecarOpts = append(sidecarOpts, sidecar.SidecarWithClient(httpClient))
	}

	service := custom.NewSidecar(sidecarOpts...)

	// subscribe by group
	for _, s := range config.Consumers {
		if len(s) == 0 {
			continue
		}

		service.ReadEventsFromBroker(s)
	}

	// create http server
	router := mux.NewRouter()

	publish := http.NewPublishHandler(service)
	state := http.NewStateHandler(service)

	router.Methods("POST").Path("/publish").HandlerFunc(publish.Handle)
	router.Methods("POST").Path("/state/{storeId}").HandlerFunc(state.HandlePost)
	router.Methods("GET").Path("/state/{storeId}").HandlerFunc(state.HandleList)
	router.Methods("GET").Path("/state/{storeId}/{key}").HandlerFunc(state.HandleGet)
	router.Methods("DELETE").Path("/state/{storeId}/{key}").HandlerFunc(state.HandleDelete)

	apiOpts := []api.ApiOption{
		api.ApiWithNamespace(config.Namespace),
		api.ApiWithName(config.Name),
		api.ApiWithVersion(config.Version),
		api.ApiWithAddress(config.HttpAddress),
	}

	httpServer := httpapi.NewApi(apiOpts...)

	httpServer.Handle("/", router)

	// create rpc server
	serverOpts := []server.ServerOption{
		server.ServerWithNamespace(config.Namespace),
		server.ServerWithName(config.Name),
		server.ServerWithVersion(config.Version),
		server.ServerWithAddress(config.RpcAddress),
	}

	grpcServer := grpcserver.NewServer(serverOpts...)

	rpc.RegisterStateHandler(
		grpcServer,
		rpc.NewStateHandler(
			service,
		),
	)

	rpc.RegisterPublishHandler(
		grpcServer,
		rpc.NewPublishHandler(
			service,
		),
	)

	// wait group and error chan
	wg := &sync.WaitGroup{}
	errCh := make(chan error, 2)

	// run rpc server
	wg.Add(1)
	go func() {
		defer wg.Done()
		errCh <- grpcServer.Run()
	}()

	// run http server
	wg.Add(1)
	go func() {
		defer wg.Done()
		errCh <- httpServer.Run()
	}()

	// block here
	err = <-errCh
	if err != nil {
		log.Errorf("failed to start sidecar: %v", err)
	}

	// unsubscribe by group
	for _, s := range config.Consumers {
		if err := service.UnsubscribeFromBroker(s); err != nil {
			log.Errorf("failed to unsubscribe from broker %s: %v", s, err)
		}
	}

	// graceful shutdown
	wait := make(chan struct{})

	go func() {
		defer close(wait)
		wg.Wait()
	}()

	select {
	case <-wait:
	case <-time.After(30 * time.Second):
	}

	log.Info("successfully stopped sidecar")
}
