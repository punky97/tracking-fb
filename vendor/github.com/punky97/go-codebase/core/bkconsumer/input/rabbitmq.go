package input

import (
	"context"
	"fmt"
	"github.com/punky97/go-codebase/core/drivers/queue"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/utils"
	core_models "github.com/punky97/go-codebase/exmsg/core/models"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

const (
	// RabbitmqMode mode
	RabbitmqMode = "rabbitmq"
)

// RabbitMQInput --
type RabbitMQInput struct {
	*queue.Consumer
	Delivery  <-chan amqp.Delivery
	XchName   string
	QueueName string
}

func preCheckInputInfo(inp *core_models.RmqInputConf) {
	if inp.Exch.Name == "" || inp.Mode == "" {
		logger.BkLog.Fatal("Invalid input information: exchange name or mode is empty")
	}
}

// BuildRabbitmqInput --
func BuildRabbitmqInput(ctx context.Context, groupName string, enableRetry bool,
	inp *core_models.RmqInputConf) (*RabbitMQInput, error) {
	preCheckInputInfo(inp)

	// create queue name if needed
	if inp.Queue.Name == "" {
		var queueName string
		var gName = groupName
		if groupName == "" {
			gName, _ = utils.GenerateRandomString(10)
			// auto delete must be set to true if group name is empty
			inp.Queue.AutoDelete = true
		}
		queueName = fmt.Sprintf("%v_%v", inp.Exch.Name, gName)
		inp.Queue.Name = queueName
	}

	uri := viper.GetString("rabbitmq.uri")
	consumer := queue.NewRMQConsumerFConfig(uri, inp, nil, enableRetry, false)

	if err := consumer.Connect(); err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error: %v", err))
		return nil, fmt.Errorf("cannot connect to rabbitmq. %v", err)
	}

	deliveries, err := consumer.AnnounceQueue()
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when calling AnnounceQueue(): %v", err.Error()))
		return nil, fmt.Errorf("cannot create or consume from exch/queue. %v. %v", inp.Exch, inp.Queue)
	}

	return &RabbitMQInput{Consumer: consumer, Delivery: deliveries, XchName: inp.Exch.Name,
		QueueName: inp.Queue.Name}, nil
}

// BuildRabbitmqInput --
func BuildDelayRabbitmqInput(dqc *core_models.DelayQueueConf) error {
	uri := viper.GetString("rabbitmq.uri")
	consumer := queue.NewRMQConsumerFConfig(uri, nil, dqc, false, true)

	if err := consumer.Connect(); err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error: %v", err))
		return fmt.Errorf("cannot connect to rabbitmq. %v", err)
	}

	_, err := consumer.AnnounceQueue()
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when calling AnnounceQueue(): %v", err.Error()))
		return err
	}

	return nil
}
