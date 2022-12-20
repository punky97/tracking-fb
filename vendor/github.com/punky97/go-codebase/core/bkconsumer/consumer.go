package bkconsumer

import (
	"context"
	"fmt"
	"github.com/punky97/go-codebase/core/bkconsumer/input"
	"github.com/punky97/go-codebase/core/bkconsumer/reporter"
	"github.com/punky97/go-codebase/core/bkconsumer/utils"
	"github.com/punky97/go-codebase/core/logger"
	reporter2 "github.com/punky97/go-codebase/core/reporter"
	"github.com/punky97/go-codebase/core/utils/codec"
	"github.com/spf13/cast"
	"github.com/streadway/amqp"
	"strings"
	"sync"
	"time"

	"sync/atomic"

	"reflect"
)

// ProcessStatus --
type ProcessStatus struct {
	Code    int64
	Message []byte
	Delay   int64 // used for ProcessForceFailReproduce only
}

var ( // Status
	// ProcessOK marks message as done
	ProcessOK = ProcessStatus{Code: utils.ProcessOKCode}
	// ProcessFailRetry marks message as fail and retry
	ProcessFailRetry = ProcessStatus{Code: utils.ProcessFailRetryCode}
	// ProcessFailDrop marks message as fail and drop
	ProcessFailDrop = ProcessStatus{Code: utils.ProcessFailDropCode}
	// ProcessFailReproduce --
	ProcessFailReproduce = ProcessStatus{Code: utils.ProcessFailReproduceCode}
	// ProduceFailForceReproduce
	ProcessFailForceReproduce = ProcessStatus{Code: utils.ProcessFailForceReproduceCode}
)

// NewProcessStatus --
func NewProcessStatus(code int64, message []byte) ProcessStatus {
	return ProcessStatus{Code: code, Message: message}
}

const (
	// DefaultConsumerName --
	DefaultConsumerName = "**_QJVtqiSTGcrFkatIyzja_**"
)

var (
	// DefaultBufferSize Default internal buffer size
	DefaultBufferSize = 1000
)

type emptyDropMsgHandler struct {
}

func (h *emptyDropMsgHandler) HandleDropMessage(int64, string, []byte) {
	return
}

func run(ctx context.Context, task *Task, name string, consumerDef ConsumerDef,
	handler HandlerWOption, wait *sync.WaitGroup) {
	defer wait.Done()
	var success, retry, drop int64
	var queueName string

	namePattern := strings.Split(name, "-")
	if len(namePattern) == 2 {
		queueName = namePattern[len(namePattern)-1]
	} else {
		queueName = namePattern[0]
	}

	if !handler.ignoreMetrics {
		tick := time.NewTicker(2 * time.Minute)
		defer tick.Stop()

		go func() {
			for {
				select {
				case <-tick.C:
					logger.BkLog.Debugf(`{"type": "consumer", "counter": {"success": %v, "fail_retried": %v, "drop": %v}}`,
						success, retry, drop)
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// drop message handler
	var dropHandler DropMessageHandler = &emptyDropMsgHandler{}
	if _, ok := handler.MsgHandler.(DropMessageHandler); ok {
		dropHandler = handler.MsgHandler.(DropMessageHandler)
	}

	// metrics
	workerMetrics := task.taskMetrics
	var rp reporter.ConsumerReporter
	rp = reporter.NewEmptyConsumerReporter()
	if !handler.ignoreMetrics && reporter2.IsMetricsEnable() {
		rp = reporter.NewConsumerReporter(reflect.TypeOf(handler.MsgHandler).Elem().PkgPath()+"."+reflect.TypeOf(handler.MsgHandler).Elem().Name(), workerMetrics)
	}

	switch consumer := consumerDef.Consumer.(type) {
	case *input.RabbitMQInput:
		consumer.Handle(consumer.Delivery, func(deliveries <-chan amqp.Delivery) {
			for {
				select {
				case d := <-deliveries:
					if len(d.Body) > 0 {
						start := time.Now()
						if isCompress, _ := d.Headers["x-compress"]; cast.ToBool(isCompress) {
							d.Body = codec.Decompress(d.Body)
						}

						resp := handler.Handle(d.Body)
						switch resp.Code {
						case utils.ProcessOKCode:
							atomic.AddInt64(&success, 1)
							d.Ack(false)
						case utils.ProcessFailDropCode:
							atomic.AddInt64(&drop, 1)
							rp.Report(d.Body)
							d.Ack(false)
							dropHandler.HandleDropMessage(resp.Code, utils.DropFail, d.Body)
						case utils.ProcessFailRetryCode:
							atomic.AddInt64(&retry, 1)
							rp.Report(d.Body)

							if isCompress, _ := d.Headers["x-compress"]; cast.ToBool(isCompress) {
								resp.Message = codec.Compress(resp.Message)
							}

							if handler.EnableRetry {
								if !d.Redelivered {
									logger.BkLog.Debugf("Retry for one time: %v", string(d.Body))
									d.Nack(false, true)
								} else {
									logger.BkLog.Warnf("Drop redelivered message: %v", string(d.Body))
									d.Ack(false)
									dropHandler.HandleDropMessage(resp.Code, utils.DropRedeliveryFail, d.Body)
								}
							} else {
								d.Nack(false, true)
							}
						case utils.ProcessFailReproduceCode, utils.ProcessFailForceReproduceCode:
							var republishMsg = d
							if resp.Message != nil && len(resp.Message) > 0 {
								republishMsg = amqp.Delivery{Headers: d.Headers, Body: resp.Message, Exchange: d.Exchange, RoutingKey: d.RoutingKey}
							}

							if !handler.EnableRetry {
								err := task.producer.PublishMessage(republishMsg)
								if err != nil {
									logger.BkLog.Warnf("Cannot republish message %v. %v", string(d.Body), err)
									d.Nack(false, true)
								} else {
									d.Ack(false)
								}
							} else {
								if republishMsg.Headers == nil {
									republishMsg.Headers = amqp.Table{}
								}

								var numRetries = 1
								if lastReties, ok := republishMsg.Headers["x-retries"]; ok {
									numRetries = cast.ToInt(lastReties)
									if resp.Code == utils.ProcessFailReproduceCode {
										numRetries += 1
									}
								}

								republishMsg.Headers["x-retries"] = fmt.Sprintf("%v", numRetries)
								if numRetries > handler.Retries && handler.Retries != -1 {
									logger.BkLog.Warnf("Drop message %v due to reach max retries", string(republishMsg.Body))
									d.Ack(false)
									dropHandler.HandleDropMessage(resp.Code, utils.DropRetryLimitExceed, d.Body)
								} else {
									if resp.Delay > 0 {
										republishMsg.Expiration = fmt.Sprintf("%v", resp.Delay)
									} else {
										republishMsg.Expiration = handler.delayTimeFunc.Process(handler.TTL, numRetries)
									}
									republishMsg.Exchange = fmt.Sprintf("%v.retry1", task.retryExch)
									republishMsg.RoutingKey = queueName
									err := task.retryProducer.PublishMessage(republishMsg)
									if err != nil {
										logger.BkLog.Warnf("Cannot republish message %v. %v", string(d.Body), err)
										d.Nack(false, true)
									} else {
										d.Ack(false)
									}
								}
							}

						default: // case of using defer in handler, unkown code
							atomic.AddInt64(&drop, 1)
							rp.Report(d.Body)
							d.Ack(false)
							dropHandler.HandleDropMessage(0, utils.DropInternalError, d.Body)
						}

						rp.End(start)
						rp.Handled(int(resp.Code))
					}
				case <-ctx.Done():
					// exit consumer
					consumer.Close()
					logger.BkLog.Debugf("Exiting %v...", name)
					return

				default:
					time.Sleep(time.Duration(handler.IdleSleepTime) * time.Millisecond)
				}
			}
		}, int(handler.Replica))
	case *input.RedisInput:
		logger.BkLog.Info("handling redis")
		consumer.Handle(consumer.Out, func(deliveries <-chan input.KV) {
			for {
				select {
				case kv := <-deliveries:
					k := kv.Key
					d := kv.Value
					if len(d) > 0 {
						start := time.Now()
						resp := handler.Handle([]byte(d))
						switch resp.Code {
						case utils.ProcessOKCode:
							atomic.AddInt64(&success, 1)
						case utils.ProcessFailDropCode:
							atomic.AddInt64(&drop, 1)
							rp.Report([]byte(d))
						case utils.ProcessFailRetryCode, utils.ProcessFailReproduceCode, utils.ProcessFailForceReproduceCode:
							atomic.AddInt64(&retry, 1)
							rp.Report([]byte(d))
							consumer.LPush(ctx, k, d)
						}

						rp.End(start)
						rp.Handled(int(resp.Code))
					}
				case <-ctx.Done():
					// exit consumer
					consumer.Close()
					logger.BkLog.Debugf("Exiting task %v...", name)
					return
				}
			}
		}, int(handler.Replica))
	}
	logger.BkLog.Infof("Exited consumer %v...", name)
}
