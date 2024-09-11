package config

import "os"

var (
	Namespace          = os.Getenv("NAMESPACE")
	Name               = os.Getenv("NAME")
	Version            = os.Getenv("VERSION")
	HttpAddress        = os.Getenv("HTTP_ADDRESS")
	RpcAddress         = os.Getenv("RPC_ADDRESS")
	ServiceName        = os.Getenv("SERVICE_NAME")
	ServicePort        = os.Getenv("SERVICE_PORT")
	ServiceProtocol    = os.Getenv("SERVICE_PROTOCOL")
	Store              = os.Getenv("STORE")
	StoreAddress       = os.Getenv("STORE_ADDRESS")
	DB                 = os.Getenv("DB")
	Stores             = Split(os.Getenv("STORES"))
	Broker             = os.Getenv("BROKER")
	BrokerAddress      = os.Getenv("BROKER_ADDRESS")
	Producers          = Split(os.Getenv("PRODUCERS"))
	Consumers          = Split(os.Getenv("CONSUMERS"))
	AwsAccessKeyId     = os.Getenv("AWS_ACCESS_KEY_ID")
	AwsSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
)
