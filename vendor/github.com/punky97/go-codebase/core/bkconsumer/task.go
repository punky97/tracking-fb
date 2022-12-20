package bkconsumer

import (
	"context"
	"fmt"
	"github.com/punky97/go-codebase/core/bkconsumer/input"
	"github.com/punky97/go-codebase/core/bkconsumer/output"
	"github.com/punky97/go-codebase/core/bkconsumer/reporter"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/utils"
	core_models "github.com/punky97/go-codebase/exmsg/core/models"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultSleepTimeWhenIdle = 1000
	defaultTTL               = 1000
	defaultRetry             = 5
)

// ConsumerDef --
type ConsumerDef struct {
	Consumer interface{}
	Mode     string
}

// ProducerDef --
type ProducerDef struct {
	Producer interface{}
	Mode     string
}

// StreamMessages for processing from multiple streams on just one handler
type StreamMessages struct {
	StreamName string
	Message    []byte
}

// MsgHandler --
type MsgHandler interface {
	Handle(msg []byte) ProcessStatus
	//HandleMultipleStream(msgs StreamMessages) ProcessStatus
}

// DropMessageHandler --
type DropMessageHandler interface {
	HandleDropMessage(int64, string, []byte)
}

// HandlerWOption --
type HandlerWOption struct {
	MsgHandler
	Replica       int64
	IdleSleepTime int
	Retries       int
	EnableRetry   bool
	RetryMode     string
	TTL           int

	delayTimeFunc waitAlgorithm
	// allow disable metrics
	ignoreMetrics bool
}

// Task --
type Task struct {
	TaskDefinition     *GConsumerDef
	consumerHandler    map[string]HandlerWOption
	consumers          map[string]ConsumerDef
	producer           *output.RabbitMQOutput // for convenience, we just need one producer for all outputs
	retryProducer      *output.RabbitMQOutput // for convenience, we just need one producer for all outputs
	ctx                context.Context
	cancelFunc         context.CancelFunc
	retryExch          string
	shutDownWg         sync.WaitGroup
	timeShutDownExpire time.Duration

	// metrics
	taskMetrics *reporter.ConsumerWorkerMetrics
}

func (t *Task) prepare() {
	t.buildIO()
}

// InitializeHandler initialize message handlers for the task
func (t *Task) InitializeHandler(handlers map[string]HandlerWOption) {
	for k, v := range handlers {
		replicaNumber := v.Replica
		if replicaNumber <= 0 {
			v.Replica = int64(runtime.NumCPU())
		}
		retries := v.Retries
		if retries <= 0 && retries != -1 {
			v.Retries = viper.GetInt("task.max_retries")
			if v.Retries <= 0 {
				v.Retries = defaultRetry
			}
		}

		if !v.EnableRetry {
			v.EnableRetry = viper.GetBool("task.enable_retry")
		}

		if v.TTL <= 0 {
			v.TTL = viper.GetInt("task.ttl")
			if v.TTL <= 0 {
				v.TTL = defaultTTL
			}
		}

		if v.RetryMode == "" {
			v.RetryMode = LinearAlgorithm
		}

		if v.EnableRetry {
			logger.BkLog.Infof("Retry mode is enable with %v mode, ttl %v, retries %v", v.RetryMode, v.TTL, v.Retries)
		}

		v.delayTimeFunc = GetWaitAlgorithm(v.RetryMode)

		v.IdleSleepTime = viper.GetInt("task.idle_sleep_time")
		if v.IdleSleepTime > 0 {
			v.IdleSleepTime = defaultSleepTimeWhenIdle
		}
		logger.BkLog.Infof("Replica of %v: %v with idle time %v (ms)", v.MsgHandler, v.Replica, v.IdleSleepTime)
		t.consumerHandler[k] = v
	}
}

func (t *Task) buildInput() {
	exchangeList := make([]string, 0)
	for _, i := range t.TaskDefinition.Inputs {
		logger.BkLog.Debug("input: ", utils.ToJSONString(t.TaskDefinition.Inputs))
		switch i.Mode {
		case input.RabbitmqMode:
			c, err := input.BuildRabbitmqInput(t.ctx,
				t.TaskDefinition.Group.Name, false, i.Config.(*core_models.RmqInputConf))
			if err != nil {
				logger.BkLog.Fatal("Cannot connect to rabbitmq", i)
				return // for compiler
			}
			key := fmt.Sprintf("%v-%v", c.XchName, c.QueueName)
			t.consumers[key] = ConsumerDef{Consumer: c, Mode: i.Mode}
			if !utils.SliceContain(exchangeList, c.XchName) {
				exchangeList = append(exchangeList, c.XchName)
			}

		case input.KafkaMode:
			logger.BkLog.Fatal("Currently not supported")
		case input.FileMode:
			logger.BkLog.Fatal("Currently not supported")
		case input.RedisMode:
			ri := i.Config.(*input.RedisInputConf)
			c := input.BuildRedisInput(t.ctx, ri)
			key := strings.Join(ri.Keys, "-")
			t.consumers[key] = ConsumerDef{Consumer: c, Mode: i.Mode}
		}
	}

	// build retry exchange
	t.retryExch = t.GenRetryExchange(exchangeList)

	// build delay queue
	if t.TaskDefinition.DelayQueue != nil && t.TaskDefinition.DelayQueue.Name != "" && len(t.TaskDefinition.DelayQueue.Output) > 0 {
		t.buildDelayQueue()
	}
}

func (t *Task) buildDelayQueue() {
	err := input.BuildDelayRabbitmqInput(t.TaskDefinition.DelayQueue)
	if err != nil {
		logger.BkLog.Fatal("Cannot connect to rabbitmq delay")
	}
}

func (t *Task) buildRetry(key string, retryMode bool, inputConf *core_models.RmqInputConf) {
	_, err := input.BuildRabbitmqInput(t.ctx,
		t.TaskDefinition.Group.Name, retryMode, inputConf)
	if err != nil {
		logger.BkLog.Fatal("Cannot connect to rabbitmq retry")
	}
}

func (t *Task) buildOutput() {
	for _, i := range t.TaskDefinition.Outputs {
		switch i.Mode {
		case output.RabbitmqMode:
			co := i.Config.(*core_models.RmqOutputConf)
			if t.producer == nil {
				t.producer = CreateOutput(co)
				t.retryProducer = CreateOutput(co)
			} else {
				_ = t.producer.DeclareSpecificExchN(co.Exch)
				_ = t.retryProducer.DeclareSpecificExchN(co.Exch)
			}
		}
	}
}

func CreateOutput(co *core_models.RmqOutputConf) *output.RabbitMQOutput {
	p, err := output.BuildRabbitmqOutput(co)
	if err != nil {
		logger.BkLog.Fatal("Cannot initialize rabbitmq producer ", err)
	}
	if err := p.DeclareSpecificExchN(co.Exch); err != nil {
		logger.BkLog.Fatal("Cannot declare exchange due to error ", err)
	}

	return p
}

func (t *Task) buildIO() {
	t.buildInput()
	t.buildOutput()
}

func (t *Task) findHandlerForConsumer(consumerName string) (HandlerWOption, error) {
	// find exactly
	logger.BkLog.Debugf("Consumer name: %v and t_consumer: %v", consumerName, t.consumerHandler)
	if v, ok := t.consumerHandler[consumerName]; ok {
		return v, nil
	}
	// find by exchange only
	for k, v := range t.consumerHandler {
		if consumerName == k {
			return v, nil
		} else if strings.Contains(consumerName, k) {
			return v, nil
		}
	}
	if _, ok := t.consumerHandler[DefaultConsumerName]; ok { // get default consumer
		return t.consumerHandler[DefaultConsumerName], nil
	}
	return HandlerWOption{}, fmt.Errorf("Cannot find handler for consumer %v", consumerName)
}

func (t *Task) start() {

	t.taskMetrics.Start()

	logger.BkLog.Debug("t_consumer: ", t.consumers)
	for k, v := range t.consumers {
		if h, err := t.findHandlerForConsumer(k); err == nil {
			logger.BkLog.Infof("Bind %v to %v", h.MsgHandler, k)
			t.shutDownWg.Add(1)
			go run(t.ctx, t, k, v, h, &t.shutDownWg)
		} else {
			logger.BkLog.Fatal("Cannot find handler for consumer ", k)
		}
	}
	t.shutDownWg.Wait()
	logger.BkLog.Info("Exiting task...")
}

func (t *Task) stop() {
	t.cancelFunc()

	c := make(chan struct{})

	go func() {
		t.shutDownWg.Wait()
		c <- struct{}{}
	}()

	select {
	case <-c:
		break
	case <-time.After(t.timeShutDownExpire):
		logger.BkLog.Warn("Graceful shutdown time out")
		break
	}

	t.taskMetrics.Close()

	if t.producer != nil {
		defer t.producer.Close()
	}
	logger.BkLog.Info("Wait for other task to complete...")
	time.Sleep(100 * time.Millisecond)
}

func (t *Task) GenRetryExchange(consumerExchs []string) string {
	if t.TaskDefinition.RetryExchange != "" {
		return t.TaskDefinition.RetryExchange
	}

	return strings.Join(consumerExchs, "-")
}
