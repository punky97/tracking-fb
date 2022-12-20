package bkconsumer

import (
	"encoding/json"
	"github.com/punky97/go-codebase/core/bkconsumer/input"
	"github.com/punky97/go-codebase/core/bkconsumer/output"
	"github.com/punky97/go-codebase/core/logger"
	core_models "github.com/punky97/go-codebase/exmsg/core/models"
)

// Input --
type Input struct {
	Mode   string
	Config interface{}
}

// Output --
type Output struct {
	Mode   string
	Config interface{}
}

// GConsumerDef --
type GConsumerDef struct {
	Type          string
	Name          string
	RetryExchange string
	Group         *core_models.ConsumerGroup
	DelayQueue    *core_models.DelayQueueConf
	Inputs        []*Input
	Outputs       []*Output
}

func parseTaskDef(def string) *GConsumerDef {
	taskDef := &struct {
		Type          string                      `json:"type,omitempty"`
		Name          string                      `json:"name,omitempty"`
		RetryExchange string                      `json:"retry_exchange,omitempty"`
		Group         *core_models.ConsumerGroup  `json:"group,omitempty"`
		DelayQueue    *core_models.DelayQueueConf `json:"delay_queue,omitempty"`
		Inputs        []interface{}               `json:"inputs,omitempty"`
		Outputs       []interface{}               `json:"outputs,omitempty"`
	}{}

	if err := json.Unmarshal([]byte(def), &taskDef); err != nil {
		logger.BkLog.Fatal("Cannot parse task info from input", def)
	}

	result := GConsumerDef{Type: taskDef.Type, Name: taskDef.Name, Group: taskDef.Group, DelayQueue: taskDef.DelayQueue, RetryExchange: taskDef.RetryExchange}
	// input
	for _, inp := range taskDef.Inputs {
		inpM, ok := inp.(map[string]interface{})
		if !ok {
			logger.BkLog.Warn("Bad input. ", inp)
			continue
		}
		t, ok := inpM["mode"]
		if !ok {
			logger.BkLog.Warn("Bad input. Mode not found", inp)
			continue
		}
		var rit interface{}
		switch t {
		case "rabbitmq":
			rit = &core_models.RmqInputConf{}
		case "redis":
			rit = &input.RedisInputConf{}
		case "kafka":
			fallthrough
		case "file":
			fallthrough
		default:
			logger.BkLog.Warn("Currently not supported. ", t)
			continue
		}

		if fillByTags(inp, rit) != nil {
			continue
		}

		result.Inputs = append(result.Inputs, &Input{Mode: t.(string), Config: rit})
	}

	// output
	for _, out := range taskDef.Outputs {
		outM, ok := out.(map[string]interface{})
		if !ok {
			logger.BkLog.Warn("Bad output. ", out)
			continue
		}
		t, ok := outM["mode"]
		if !ok {
			logger.BkLog.Warn("Bad input. Mode not found", out)
			continue
		}
		var rit interface{}
		switch t {
		case "rabbitmq":
			rit = &core_models.RmqOutputConf{}
		case "redis":
			rit = &output.RedisOutputConf{}
		case "kafka":
			fallthrough
		case "file":
			fallthrough
		default:
			logger.BkLog.Warn("Currently not supported. ", t)
			continue
		}

		if fillByTags(out, rit) != nil {
			continue
		}

		result.Outputs = append(result.Outputs, &Output{Mode: t.(string), Config: rit})
	}

	return &result
}

func fillByTags(i interface{}, o interface{}) error {
	b, _ := json.Marshal(i)
	err := json.Unmarshal(b, o)
	return err
}
