package reporter

import (
	"errors"
	"fmt"
	"github.com/punky97/go-codebase/core/drivers/queue"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/reporter/constant"
	"github.com/punky97/go-codebase/core/reporter/metrics"
	"github.com/punky97/go-codebase/core/utils"
	"github.com/spf13/viper"
	"math"
	"runtime"
	"strings"
)

var (
	rabbitUri string
	retries   int
)

func GetAPIDMSAlertConfig(appName string) []constant.AlertConfig {
	normAppName := strings.Join(utils.StringSlice(appName, "/"), ".")
	configName := "statistics.alert." + normAppName
	m := viper.GetStringMap(configName)

	configs := make([]constant.AlertConfig, 0)
	for name, conf := range m {
		if alertConf, ok := conf.(map[string]interface{}); ok {
			if requestRange, ok := alertConf["request_range"].(map[string]interface{}); ok {
				alertConfig := constant.AlertConfig{
					Name:        name,
					ErrorRate:   alertConf["error_rate"].(float64),
					RequestFrom: int64(requestRange["from"].(int64)),
					RequestTo:   int64(requestRange["to"].(int64)),
				}

				if alertConfig.RequestTo < 0 {
					alertConfig.RequestTo = math.MaxInt64
				}

				configs = append(configs, alertConfig)
			}
		}
	}

	return configs
}

func GetProcStats() metrics.ProcStats {
	stats := metrics.ProcStats{}
	stats.NumGoRoutine = runtime.NumGoroutine()
	// TODO: re-implement cpu,mem metrics
	return stats
}

func IsMetricsEnable() bool {
	return viper.GetBool("reporter.enable")
}

func IsNotifyEnable() map[string]interface{} {
	return viper.GetStringMap("notifier")
}

func IsAllMetricsEnable() bool {
	return viper.GetBool("reporter.enable_all")
}

type RProducer struct {
	*queue.Producer
}

func GetProducerFromConf() {
	rabbitUri = viper.GetString("rabbitmq.uri")
	retries = viper.GetInt("rabbitmq.retries")
}

func PublishSimple(exchName string, data []byte) (err error) {
	if len(rabbitUri) > 0 {
		p := queue.NewRMQProducerFromConf(&queue.RabbitMqConfiguration{
			URI: rabbitUri,
		}, 1, retries)

		if err := p.ConnectMulti(1); err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("Cannot connect to rabbitmq. Please check configuration file for more information: %v", err))
			return err
		}

		p.Start()
		if err := p.PublishSimple(exchName, data); err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("Error when publish: %v", err), "data", string(data))
		}
		p.Close()

		return err
	}

	return nil
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
