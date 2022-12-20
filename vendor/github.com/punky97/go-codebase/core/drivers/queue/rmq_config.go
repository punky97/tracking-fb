package queue

import (
	"github.com/punky97/go-codebase/core/utils"
	core_models "github.com/punky97/go-codebase/exmsg/core/models"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

// RmqConfiguration is configuration for exchange
type RmqConfiguration struct {
	Name       string
	Type       string
	AutoDelete bool
	Durable    bool
	Internal   bool
	Exclusive  bool
	NoWait     bool
	Others     map[string]interface{}
}

func censorURI(uri string) string {
	rbUri, err := amqp.ParseURI(uri)
	if err != nil {
		return ""
	}

	rbUri.Password = utils.CensorString(rbUri.Password)
	return rbUri.String()
}

// RabbitMqConfiguration configuration for rabbitmq
type RabbitMqConfiguration struct {
	// amqp://user:pass@host:port/vhost?heartbeat=10&connection_timeout=10000&channel_max=100
	URI string
	// Exchange configuration
	ExchangeConfig RmqConfiguration
	QueueConfig    RmqConfiguration
	DelayConfig    core_models.DelayQueueConf
	EnableRetry    bool
	EnableDelay    bool
}

// DefaultRmqConfiguration create default config for rabbitmq client
func DefaultRmqConfiguration(uri, exchangeName, queueName string) *RabbitMqConfiguration {
	return &RabbitMqConfiguration{
		uri,
		RmqConfiguration{exchangeName, "direct", false, true, false, false, false, make(map[string]interface{})},
		RmqConfiguration{queueName, "direct", false, true, false, false, false, make(map[string]interface{})},
		core_models.DelayQueueConf{},
		false,
		false,
	}
}

// DefaultRMqConfFromConfig load from default configuration
func DefaultRMqConfFromConfig() *RabbitMqConfiguration {
	uri := viper.GetString("rabbitmq.uri")
	exch := viper.GetString("rabbitmq.exchange")
	exchType := viper.GetString("rabbitmq.exchange_type")
	xchAutoDelete := viper.GetBool("rabbitmq.exchange_autodelete")
	xchDurable := viper.GetBool("rabbitmq.exchange_durable")

	// queue
	queue := viper.GetString("rabbitmq.queue")
	qType := viper.GetString("rabbitmq.queue_type")
	qAutoDelete := viper.GetBool("rabbitmq.queue_autodelete")
	qDurable := viper.GetBool("rabbitmq.queue_durable")

	exchConf := RmqConfiguration{exch, exchType, xchAutoDelete, xchDurable, false, false, false, make(map[string]interface{})}
	queueConf := RmqConfiguration{queue, qType, qAutoDelete, qDurable, false, false, false, make(map[string]interface{})}
	return &RabbitMqConfiguration{
		URI:            uri,
		ExchangeConfig: exchConf,
		QueueConfig:    queueConf,
	}
}

// LoadQueueConfig Load queue configuration from file
func LoadQueueConfig() *RmqConfiguration {

	// queue
	queue := viper.GetString("rabbitmq.queue")
	qType := viper.GetString("rabbitmq.queue_type")
	qAutoDelete := viper.GetBool("rabbitmq.queue_autodelete")
	qDurable := viper.GetBool("rabbitmq.queue_durable")
	return &RmqConfiguration{queue, qType, qAutoDelete, qDurable, false, false, false, make(map[string]interface{})}
}

// LoadExchConfig Load exchange configuration from file
func LoadExchConfig() *RmqConfiguration {
	exch := viper.GetString("rabbitmq.exchange")
	exchType := viper.GetString("rabbitmq.exchange_type")
	xchAutoDelete := viper.GetBool("rabbitmq.exchange_autodelete")
	xchDurable := viper.GetBool("rabbitmq.exchange_durable")

	return &RmqConfiguration{exch, exchType, xchAutoDelete, xchDurable, false, false, false, make(map[string]interface{})}
}

// LoadOtherConfigParams -- internal queue size, number of retries
func LoadOtherConfigParams() map[string]interface{} {
	internalQueueSize := viper.GetInt("rabbitmq.internal_queue_size")
	retries := viper.GetInt("rabbitmq.retries")
	uri := viper.GetString("rabbitmq.uri")
	numThread := viper.GetInt("rabbitmq.num_thread")
	m := make(map[string]interface{})
	m["internal_queue_size"] = internalQueueSize
	m["retries"] = retries
	m["uri"] = uri
	m["num_thread"] = numThread
	return m
}
