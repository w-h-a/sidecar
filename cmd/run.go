package cmd

import (
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/urfave/cli"
	"github.com/w-h-a/pkg/broker"
	"github.com/w-h-a/pkg/client/grpcclient"
	"github.com/w-h-a/pkg/client/httpclient"
	"github.com/w-h-a/pkg/serverv2"
	grpcserver "github.com/w-h-a/pkg/serverv2/grpc"
	httpserver "github.com/w-h-a/pkg/serverv2/http"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/sidecar/custom"
	"github.com/w-h-a/pkg/store"
	"github.com/w-h-a/pkg/telemetry/log"
	"github.com/w-h-a/sidecar/cmd/config"
	"github.com/w-h-a/sidecar/cmd/grpc"
	"github.com/w-h-a/sidecar/cmd/http"
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
	_, grpcPort, _ := strings.Cut(config.GrpcAddress, ":")

	sidecarOpts := []sidecar.SidecarOption{
		sidecar.SidecarWithServiceName(config.ServiceName),
		sidecar.SidecarWithHttpPort(sidecar.Port{Port: httpPort}),
		sidecar.SidecarWithGrpcPort(sidecar.Port{Port: grpcPort}),
		sidecar.SidecarWithServicePort(sidecar.Port{Port: config.ServicePort, Protocol: config.ServiceProtocol}),
		sidecar.SidecarWithStores(stores),
		sidecar.SidecarWithBrokers(brokers),
	}

	if config.ServiceProtocol == "grpc" {
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

	// base server opts
	opts := []serverv2.ServerOption{
		serverv2.ServerWithNamespace(config.Namespace),
		serverv2.ServerWithName(config.Name),
		serverv2.ServerWithVersion(config.Version),
	}

	// create http server
	router := mux.NewRouter()

	httpHealth := http.NewHealthHandler()
	httpPublish := http.NewPublishHandler(service)
	httpState := http.NewStateHandler(service)

	router.Methods("GET").Path("/health/check").HandlerFunc(httpHealth.Check)
	router.Methods("POST").Path("/publish").HandlerFunc(httpPublish.Handle)
	router.Methods("POST").Path("/state/{storeId}").HandlerFunc(httpState.HandlePost)
	router.Methods("GET").Path("/state/{storeId}").HandlerFunc(httpState.HandleList)
	router.Methods("GET").Path("/state/{storeId}/{key}").HandlerFunc(httpState.HandleGet)
	router.Methods("DELETE").Path("/state/{storeId}/{key}").HandlerFunc(httpState.HandleDelete)

	httpOpts := []serverv2.ServerOption{
		serverv2.ServerWithAddress(config.HttpAddress),
	}

	httpOpts = append(httpOpts, opts...)

	httpServer := httpserver.NewServer(httpOpts...)

	httpServer.Handle(router)

	// create grpc server
	grpcOpts := []serverv2.ServerOption{
		serverv2.ServerWithAddress(config.GrpcAddress),
	}

	grpcOpts = append(grpcOpts, opts...)

	grpcServer := grpcserver.NewServer(grpcOpts...)

	grpcHealth := grpc.NewHealthHandler()
	grpcPublish := grpc.NewPublishHandler(service)
	grpcState := grpc.NewStateHandler(service)

	grpcServer.Handle(grpcserver.NewHandler(grpcHealth))
	grpcServer.Handle(grpcserver.NewHandler(grpcPublish))
	grpcServer.Handle(grpcserver.NewHandler(grpcState))

	// wait group and error chan
	wg := &sync.WaitGroup{}
	errCh := make(chan error, 2)

	// run grpc server
	wg.Add(1)
	go func() {
		defer wg.Done()
		errCh <- grpcServer.Start()
	}()

	// run http server
	wg.Add(1)
	go func() {
		defer wg.Done()
		errCh <- httpServer.Start()
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
