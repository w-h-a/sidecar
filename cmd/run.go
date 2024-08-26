package cmd

import (
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/urfave/cli"
	"github.com/w-h-a/action/cmd/config"
	"github.com/w-h-a/action/cmd/controllers"
	"github.com/w-h-a/action/cmd/handlers"
	"github.com/w-h-a/pkg/api"
	"github.com/w-h-a/pkg/api/httpapi"
	"github.com/w-h-a/pkg/broker"
	memorybroker "github.com/w-h-a/pkg/broker/memory"
	"github.com/w-h-a/pkg/broker/snssqs"
	"github.com/w-h-a/pkg/client/grpcclient"
	"github.com/w-h-a/pkg/client/httpclient"
	"github.com/w-h-a/pkg/server"
	"github.com/w-h-a/pkg/server/grpcserver"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/sidecar/custom"
	"github.com/w-h-a/pkg/store"
	"github.com/w-h-a/pkg/store/cockroach"
	memorystore "github.com/w-h-a/pkg/store/memory"
	"github.com/w-h-a/pkg/telemetry/log"
)

func run(ctx *cli.Context) {
	// get clients
	httpClient := httpclient.NewClient()

	grpcClient := grpcclient.NewClient()

	stores := map[string]store.Store{}

	brokers := map[string]broker.Broker{}

	if len(config.Memory) > 0 {
		for _, s := range config.Stores {
			if len(s) == 0 {
				continue
			}

			stores[s] = memorystore.NewStore(
				store.StoreWithTable(s),
			)
		}

		for _, s := range config.Consumers {
			if len(s) == 0 {
				continue
			}

			publishOptions := broker.NewPublishOptions(
				broker.PublishWithTopic(s),
			)

			subscribeOptions := broker.NewSubscribeOptions(
				broker.SubscribeWithGroup(s),
			)

			brokers[s] = memorybroker.NewBroker(
				broker.BrokerWithPublishOptions(publishOptions),
				broker.BrokerWithSubscribeOptions(subscribeOptions),
			)
		}
	} else {
		for _, s := range config.Stores {
			if len(s) == 0 {
				continue
			}

			stores[s] = cockroach.NewStore(
				store.StoreWithNodes(config.StoreAddress),
				store.StoreWithDatabase(config.ServiceName),
				store.StoreWithTable(s),
			)
		}

		for _, s := range config.Producers {
			if len(s) == 0 {
				continue
			}

			publishOptions := broker.NewPublishOptions(
				broker.PublishWithTopic(s),
			)

			brokers[s] = snssqs.NewBroker(
				broker.BrokerWithPublishOptions(publishOptions),
			)
		}

		for _, s := range config.Consumers {
			if len(s) == 0 {
				continue
			}

			subscribeOptions := broker.NewSubscribeOptions(
				broker.SubscribeWithGroup(s),
			)

			brokers[s] = snssqs.NewBroker(
				broker.BrokerWithSubscribeOptions(subscribeOptions),
			)
		}
	}

	// get services
	_, httpPort, _ := strings.Cut(config.HttpAddress, ":")
	_, rpcPort, _ := strings.Cut(config.RpcAddress, ":")

	action := custom.NewSidecar(
		sidecar.SidecarWithServiceName(config.ServiceName),
		sidecar.SidecarWithHttpPort(sidecar.Port{Port: httpPort}),
		sidecar.SidecarWithRpcPort(sidecar.Port{Port: rpcPort}),
		sidecar.SidecarWithServicePort(sidecar.Port{Port: config.ServicePort, Protocol: config.ServiceProtocol}),
		sidecar.SidecarWithHttpClient(httpClient),
		sidecar.SidecarWithRpcClient(grpcClient),
		sidecar.SidecarWithStores(stores),
		sidecar.SidecarWithBrokers(brokers),
	)

	// subscribe by group
	for _, s := range config.Consumers {
		if len(s) == 0 {
			continue
		}

		action.ReadEventsFromBroker(s)
	}

	// create http server
	router := mux.NewRouter()

	publish := handlers.NewPublishHandler(action)
	state := handlers.NewStateHandler(action)

	router.Methods("POST").Path("/publish").HandlerFunc(publish.Handle)
	router.Methods("POST").Path("/state/{storeId}").HandlerFunc(state.HandlePost)
	router.Methods("GET").Path("/state/{storeId}/{key}").HandlerFunc(state.HandleGet)

	opts := []api.ApiOption{
		api.ApiWithNamespace(config.Namespace),
		api.ApiWithName(config.Name),
		api.ApiWithVersion(config.Version),
		api.ApiWithAddress(config.HttpAddress),
	}

	httpServer := httpapi.NewApi(opts...)

	httpServer.Handle("/", router)

	// create rpc server
	grpcServer := grpcserver.NewServer(
		server.ServerWithNamespace(config.Namespace),
		server.ServerWithName(config.Name),
		server.ServerWithVersion(config.Version),
		server.ServerWithAddress(config.RpcAddress),
	)

	controllers.RegisterStateController(
		grpcServer,
		controllers.NewStateController(
			action,
		),
	)

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
	err := <-errCh
	if err != nil {
		log.Errorf("failed to start action: %v", err)
	}

	// unsubscribe by group
	for _, s := range config.Consumers {
		if err := action.UnsubscribeFromBroker(s); err != nil {
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

	log.Info("successfully stopped action")
}
