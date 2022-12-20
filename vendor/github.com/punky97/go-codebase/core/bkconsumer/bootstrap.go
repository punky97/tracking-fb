package bkconsumer

import (
	"github.com/punky97/go-codebase/core/logger"
	core_models "github.com/punky97/go-codebase/exmsg/core/models"
	"time"
)

// Bootstrapping --
func Bootstrapping(taskDef string, initFunc Callback) {
	w := NewWorker(nil)
	initFunc(w)
	w.PrepareTask(taskDef)

	if Bootstrapper.F != nil && len(Bootstrapper.F) > 0 {
		for count, ff := range Bootstrapper.F {
			f := ff
			f.Init(w)
			defer f.Close(w)

			hs := f.GetHandlers()
			w.PrepareHandlers(hs)

			dmsCodes := f.GetDmsNames()
			if dmsCodes != nil && len(dmsCodes) > 0 {
				dmsClients := w.GetDmsClient(dmsCodes...)
				if len(dmsClients) > 0 {
					f.SetDmsClient(dmsClients)
				}
			}

			if p := w.GetProducer(); p != nil {
				f.SetProducer(p)
			}

			// retry exch
			for k := range hs {
				if w.task.consumerHandler[k].EnableRetry {
					inputConf := w.task.TaskDefinition.Inputs[count].Config.(*core_models.RmqInputConf)
					inputConf.Exch.Name = w.task.retryExch
					w.task.buildRetry(k, w.task.consumerHandler[k].EnableRetry, inputConf)
				}
			}
		}
	} else {
		logger.BkLog.Fatal("Cannot initialize an empty worker.")
	}

	time.Sleep(2 * time.Second)
	w.Walk()
}

// BootstrapScheduler --
func BootstrapScheduler(initFunc Callback) {
	w := NewWorker(nil)
	initFunc(w)
	w.prepareScheduler()
	p := w.GetProducerFromConf()
	if p != nil {
		w.scheduler.schedulerMetrics.AddWatchingProducer(p)
	}
	if Bootstrapper.S != nil && len(Bootstrapper.S) > 0 {
		for _, ff := range Bootstrapper.S {
			f := ff
			f.Init(w)
			defer f.Close(w)

			hs := f.GetHandlers()
			w.PrepareScheduleTask(hs)

			dmsCodes := f.GetDmsNames()
			if dmsCodes != nil && len(dmsCodes) > 0 {
				dmsClients := w.GetDmsClient(dmsCodes...)
				if len(dmsClients) > 0 {
					f.SetDmsClient(dmsClients)
				}
			}

			if p != nil {
				f.SetProducer(p)
			}
		}
	} else {
		logger.BkLog.Fatal("Cannot initialize an empty task.")
	}
	time.Sleep(2 * time.Second)
	w.Schedule()
}
