package config

import "os"

var (
	Namespace       = os.Getenv("NAMESPACE")
	Name            = os.Getenv("NAME")
	Version         = os.Getenv("VERSION")
	HttpAddress     = os.Getenv("HTTP_ADDRESS")
	RpcAddress      = os.Getenv("RPC_ADDRESS")
	StoreAddress    = os.Getenv("STORE_ADDRESS")
	BrokerAddress   = os.Getenv("BROKER_ADDRESS")
	Store           = os.Getenv("STORE")
	Stores          = Split(os.Getenv("STORES"))
	Broker          = os.Getenv("BROKER")
	Producers       = Split(os.Getenv("PRODUCERS"))
	Consumers       = Split(os.Getenv("CONSUMERS"))
	ServiceName     = os.Getenv("SERVICE_NAME")
	ServicePort     = os.Getenv("SERVICE_PORT")
	ServiceProtocol = os.Getenv("SERVICE_PROTOCOL")
	Memory          = os.Getenv("MEMORY")
)
