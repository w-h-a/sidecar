package cmd

import (
	"fmt"

	"github.com/w-h-a/pkg/broker"
	memorybroker "github.com/w-h-a/pkg/broker/memory"
	"github.com/w-h-a/pkg/broker/snssqs"
	"github.com/w-h-a/pkg/security/secret"
	"github.com/w-h-a/pkg/security/secret/env"
	"github.com/w-h-a/pkg/security/secret/ssm"
	"github.com/w-h-a/pkg/store"
	"github.com/w-h-a/pkg/store/cockroach"
	memorystore "github.com/w-h-a/pkg/store/memory"
	"github.com/w-h-a/pkg/telemetry/traceexporter"
	memorytraceexporter "github.com/w-h-a/pkg/telemetry/traceexporter/memory"
	"github.com/w-h-a/pkg/utils/memoryutils"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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

	defaultSecrets = map[string]func(...secret.SecretOption) secret.Secret{
		"ssm": ssm.NewSecret,
		"env": env.NewSecret,
	}

	defaultTraceExporters = map[string]func(...traceexporter.ExporterOption) sdktrace.SpanExporter{
		// "otelp": otelp.NewExporter,
		"memory": memorytraceexporter.NewExporter,
	}
)

func GetStoreBuilder(s string) (func(...store.StoreOption) store.Store, error) {
	storeBuilder, exists := defaultStores[s]
	if !exists && len(s) > 0 {
		return nil, fmt.Errorf("store %s is not supported", s)
	} else if !exists {
		return nil, nil
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

func GetSecretBuilder(s string) (func(...secret.SecretOption) secret.Secret, error) {
	secretBuilder, exists := defaultSecrets[s]
	if !exists && len(s) > 0 {
		return nil, fmt.Errorf("secret store %s is not supported", s)
	} else if !exists {
		return nil, nil
	}
	return secretBuilder, nil
}

func MakeSecret(secretBuilder func(...secret.SecretOption) secret.Secret, nodes []string, prefix string) secret.Secret {
	return secretBuilder(
		secret.SecretWithNodes(nodes...),
		secret.SecretWithPrefix(prefix),
	)
}

func GetTraceExporterBuilder(s string) (func(...traceexporter.ExporterOption) sdktrace.SpanExporter, error) {
	traceExporterBuilder, exists := defaultTraceExporters[s]
	if !exists && len(s) > 0 {
		return nil, fmt.Errorf("trace exporter %s is not supported", s)
	} else if !exists {
		return memorytraceexporter.NewExporter, nil
	}
	return traceExporterBuilder, nil
}

func MakeTraceExporter(tracerExporterBuilder func(...traceexporter.ExporterOption) sdktrace.SpanExporter, buffer *memoryutils.Buffer) sdktrace.SpanExporter {
	return tracerExporterBuilder(
		traceexporter.ExporterWithBuffer(buffer),
	)
}

func GetBrokerBuilder(s string) (func(...broker.BrokerOption) broker.Broker, error) {
	brokerBuilder, exists := defaultBrokers[s]
	if !exists && len(s) > 0 {
		return nil, fmt.Errorf("broker %s is not supported", s)
	} else if !exists {
		return nil, nil
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
