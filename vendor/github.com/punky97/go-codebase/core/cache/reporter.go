package cache

import (
	"github.com/punky97/go-codebase/core/drivers/queue"
	"github.com/punky97/go-codebase/core/logger"
	"errors"
	"fmt"
	"github.com/spf13/viper"
)

var (
	rabbitUri string
	retries   int
)

func GetProducerFromConf() {
	rabbitUri = viper.GetString("rabbitmq.uri")
	retries = viper.GetInt("rabbitmq.retries")
}

func PublishWithRoutingKey(exchName, routingKey string, data []byte) (err error) {
	if len(rabbitUri) > 0 {
		p := queue.NewRMQProducerFromConf(&queue.RabbitMqConfiguration{
			URI: rabbitUri,
		}, 1, retries)

		if err := p.ConnectMulti(1); err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("Cannot connect to rabbitmq. Please check configuration file for more information: %v", err))
			return err
		}

		p.Start()
		if err := p.PublishRouting(exchName, routingKey, data); err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("Error when publish to routing `%v` %v", routingKey, err), "data", string(data))
		}
		p.Close()

		return err
	} else {
		return errors.New("rabbitMQ uri is invalid.")
	}
}
