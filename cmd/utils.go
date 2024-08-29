package cmd

import (
	"fmt"

	"github.com/w-h-a/pkg/broker"
	memorybroker "github.com/w-h-a/pkg/broker/memory"
	"github.com/w-h-a/pkg/broker/snssqs"
	"github.com/w-h-a/pkg/store"
	"github.com/w-h-a/pkg/store/cockroach"
	memorystore "github.com/w-h-a/pkg/store/memory"
)

var (
	defaultStores = map[string]func(...store.StoreOption) store.Store{
		"cockroach": cockroach.NewStore,
		"memory":    memorystore.NewStore,
	}

	defaultBrokers = map[string]func(...broker.BrokerOption) broker.Broker{
		"snssqs": snssqs.NewBroker,
		"memory": memorybroker.NewBroker,
	}
)

func GetStoreBuilder(s string) (func(...store.StoreOption) store.Store, error) {
	storeBuilder, exists := defaultStores[s]
	if !exists {
		return nil, fmt.Errorf("store %s is not supported", s)
	}
	return storeBuilder, nil
}

func MakeStore(storeBuilder func(...store.StoreOption) store.Store, nodes []string, database, table string) store.Store {
	return storeBuilder(
		store.StoreWithNodes(nodes...),
		store.StoreWithDatabase(database),
		store.StoreWithTable(table),
	)
}

func GetBrokerBuilder(s string) (func(...broker.BrokerOption) broker.Broker, error) {
	brokerBuilder, exists := defaultBrokers[s]
	if !exists {
		return nil, fmt.Errorf("broker %s is not supported", s)
	}
	return brokerBuilder, nil
}

func MakeProducer(brokerBuilder func(...broker.BrokerOption) broker.Broker, nodes []string, topic string) broker.Broker {
	options := broker.NewPublishOptions(
		broker.PublishWithTopic(topic),
	)

	return brokerBuilder(
		broker.BrokerWithNodes(nodes...),
		broker.BrokerWithPublishOptions(&options),
	)
}

func MakeConsumer(brokerBuilder func(...broker.BrokerOption) broker.Broker, nodes []string, group string, memory bool) broker.Broker {
	subOptions := broker.NewSubscribeOptions(
		broker.SubscribeWithGroup(group),
	)

	if memory {
		pubOptions := broker.NewPublishOptions(
			broker.PublishWithTopic(group),
		)

		return brokerBuilder(
			broker.BrokerWithPublishOptions(&pubOptions),
			broker.BrokerWithSubscribeOptions(&subOptions),
		)
	}

	return brokerBuilder(
		broker.BrokerWithNodes(nodes...),
		broker.BrokerWithSubscribeOptions(&subOptions),
	)
}
