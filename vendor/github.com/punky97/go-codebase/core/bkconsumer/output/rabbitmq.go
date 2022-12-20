package output

import (
	"github.com/punky97/go-codebase/core/drivers/queue"
	"github.com/punky97/go-codebase/core/logger"
	core_models "github.com/punky97/go-codebase/exmsg/core/models"
	"fmt"

	"github.com/spf13/viper"
)

const (
	// RabbitmqMode mode
	RabbitmqMode = "rabbitmq"
)

// RabbitMQOutput --
type RabbitMQOutput struct {
	*queue.Producer
}

func preCheckOutputInfo(out *core_models.RmqOutputConf) {
	if out.Exch.Name == "" || out.Mode == "" {
		logger.BkLog.Fatal("Invalid output information: exchange name or mode is empty")
	}
}

// BuildRabbitmqOutput --
func BuildRabbitmqOutput(out *core_models.RmqOutputConf) (*RabbitMQOutput, error) {
	preCheckOutputInfo(out)

	uri := viper.GetString("rabbitmq.uri")
	retries := viper.GetInt64("rabbitmq.retries")
	internalQueueSize := viper.GetInt64("rabbitmq.internal_queue_size")
	numThread := viper.GetInt("rabbitmq.num_thread")
	producer := queue.NewRMQProducerFConfig(uri, retries, internalQueueSize, out)

	if numThread <= 0 {
		numThread = queue.MaxThread
	}

	if err := producer.ConnectMulti(numThread); err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Cannot connect to rabbitmq due to error: %v", err))
		return nil, fmt.Errorf("cannot connect to rabbitmq. %v", err)
	}

	producer.Start()

	return &RabbitMQOutput{Producer: producer}, nil
}
