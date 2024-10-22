package cmd

import (
	"fmt"
	"strings"

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
	otelp "github.com/w-h-a/pkg/telemetry/traceexporter/otelp"
	"github.com/w-h-a/pkg/utils/memoryutils"
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

	defaultTraceExporters = map[string]func(...traceexporter.ExporterOption) traceexporter.TraceExporter{
		"otelp":  otelp.NewExporter,
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

func GetTraceExporterBuilder(s string) (func(...traceexporter.ExporterOption) traceexporter.TraceExporter, error) {
	traceExporterBuilder, exists := defaultTraceExporters[s]
	if !exists && len(s) > 0 {
		return nil, fmt.Errorf("trace exporter %s is not supported", s)
	} else if !exists {
		return memorytraceexporter.NewExporter, nil
	}
	return traceExporterBuilder, nil
}

func MakeTraceExporter(tracerExporterBuilder func(...traceexporter.ExporterOption) traceexporter.TraceExporter, buffer *memoryutils.Buffer, nodes []string, protocol, secure string, pairs []string) traceexporter.TraceExporter {
	opts := []traceexporter.ExporterOption{
		traceexporter.ExporterWithBuffer(buffer),
		traceexporter.ExporterWithNodes(nodes...),
		traceexporter.ExporterWithProtocol(protocol),
	}

	if len(secure) > 0 {
		opts = append(opts, traceexporter.ExporterWithSecure())
	}

	headers := map[string]string{}

	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 {
			continue
		}
		headers[kv[0]] = kv[1]
	}

	opts = append(opts, traceexporter.ExporterWithHeaders(headers))

	return tracerExporterBuilder(opts...)
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
