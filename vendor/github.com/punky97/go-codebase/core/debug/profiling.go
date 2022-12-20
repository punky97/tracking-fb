package debug

import (
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/utils"
	"fmt"
	"github.com/pborman/uuid"
	"io/ioutil"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"
)

var DefaultProfileSetting = NewProfileSetting()

type profileData struct {
	cpuProfile     string
	memProfile     string
	routineProfile string
	blockProfile   string
}

type ProfilingSetting struct {
	sync.RWMutex
	closed    chan struct{}
	start     chan struct{}
	stop      chan chan []string
	isStart   bool
	limitTime int64
	name      string
}

func NewProfileSetting() *ProfilingSetting {
	var limit = int64(30) * 60000

	return &ProfilingSetting{
		closed:    make(chan struct{}),
		start:     make(chan struct{}),
		stop:      make(chan chan []string),
		isStart:   false,
		limitTime: limit,
	}
}

func EnableProfiling(id string) {
	DefaultProfileSetting.EnableProfiling(id)
}

func StartProfiling() {
	DefaultProfileSetting.Start()
}

func StopProfiling() {
	if DefaultProfileSetting.IsStarted() {
		DefaultProfileSetting.Stop()
	}
}

func (p *ProfilingSetting) EnableProfiling(name string) {
	p.name = name

	streamChan := make(chan profileData, 10)
	go initProfilingHandler(p, streamChan)
	go initProfilingStream(p, streamChan)
}

func (p *ProfilingSetting) IsStarted() bool {
	p.RLock()
	defer p.RUnlock()
	return p.isStart
}

func (p *ProfilingSetting) SetStart(started bool) {
	p.Lock()
	defer p.Unlock()
	p.isStart = started
}

func (p *ProfilingSetting) Start() {
	go func() {
		select {
		case p.start <- struct{}{}:
		case <-time.After(time.Second):
			logger.BkLog.Infof("Request Start profiling timeout")
		}
	}()
}

func (p *ProfilingSetting) Stop() []string {
	res := make(chan []string)
	select {
	case p.stop <- res:
		select {
		case m := <-res:
			return m
		case <-time.After(time.Second * 30):
			return []string{}
		}
	case <-time.After(time.Second):
		logger.BkLog.Infof("Request Stop profiling timeout")
		return []string{}
	}
}

func (p *ProfilingSetting) Close() {
	p.Lock()
	defer p.Unlock()
	close(p.closed)
}

func initProfilingHandler(profiling *ProfilingSetting, streamChan chan profileData) {
	var tm *time.Timer
	var tmChan <-chan time.Time
	var cpuProfileFile, memProfileFile, routineProfileFile, blockProfileFile *os.File
	var err error
	logger.BkLog.Infof("Init profiling handler")
	for {
		select {
		case <-profiling.closed:
			return
		case <-profiling.start:
			if profiling.IsStarted() {
				break
			}

			// start profiling
			cpuProfileFile, err = ioutil.TempFile("", "cpuprofile.pb.gz")
			if err != nil {
				logger.BkLog.Errorw(fmt.Sprintf("Cannot create cpuprofile file: %v", err))
				profiling.SetStart(false)
			} else {
				logger.BkLog.Infof("Start profiling")
				err := pprof.StartCPUProfile(cpuProfileFile)
				if err != nil {
					logger.BkLog.Errorw(fmt.Sprintf("Error when start cpu profiling: %v", err))
				}

				runtime.SetBlockProfileRate(1)
				tm = time.NewTimer(time.Millisecond * time.Duration(profiling.limitTime))
				tmChan = tm.C
				profiling.SetStart(true)
			}
		case och := <-profiling.stop:
			if !profiling.IsStarted() {
				break
			}

			logger.BkLog.Infof("Stopping Profiling")
			if !tm.Stop() {
				logger.BkLog.Infof("Profiling has been stop by limit time %v", profiling.limitTime)
			}
			// collect cpu
			pprof.StopCPUProfile()
			logger.BkLog.Infof("cpu profile saved to %v", cpuProfileFile.Name())

			// collect mem
			pHeap := pprof.Lookup("heap")
			memProfileFile, err = ioutil.TempFile("", "heap.pb.gz")
			if err != nil {
				logger.BkLog.Errorw(fmt.Sprintf("Could not create heapProfile file: %v", err))
			} else {
				if err := pHeap.WriteTo(memProfileFile, 0); err == nil {
					logger.BkLog.Infof("mem profile saved to %v", memProfileFile.Name())
				} else {
					logger.BkLog.Errorw(fmt.Sprintf("Error when dump mem profile: %v", err))
				}
			}

			// collect routine
			pRoutine := pprof.Lookup("goroutine")
			routineProfileFile, err = ioutil.TempFile("", "routine.pb.gz")
			if err != nil {
				logger.BkLog.Errorw(fmt.Sprintf("Could not create routineProfile file: %v", err))
			} else {
				if err := pRoutine.WriteTo(routineProfileFile, 0); err == nil {
					logger.BkLog.Infof("routine profile saved to %v", routineProfileFile.Name())
				} else {
					logger.BkLog.Errorw(fmt.Sprintf("Error when dump go routine profile: %v", err))
				}
			}

			// collect block
			pBlock := pprof.Lookup("block")
			blockProfileFile, err = ioutil.TempFile("", "block.pb.gz")
			if err != nil {
				logger.BkLog.Errorw(fmt.Sprintf("Could not create blockProfile file: %v", err))
			} else {
				if err := pBlock.WriteTo(blockProfileFile, 0); err == nil {
					logger.BkLog.Infof("block profile saved to %v", blockProfileFile.Name())
				} else {
					logger.BkLog.Errorw(fmt.Sprintf("Error when dump go block profile: %v", err))
				}
			}

			data := profileData{
				cpuProfile:     cpuProfileFile.Name(),
				memProfile:     memProfileFile.Name(),
				routineProfile: routineProfileFile.Name(),
				blockProfile:   blockProfileFile.Name(),
			}
			cpuProfileFile.Close()
			memProfileFile.Close()
			routineProfileFile.Close()
			blockProfileFile.Close()

			// turn off block profile
			runtime.SetBlockProfileRate(0)

			if och != nil {
				och <- []string{data.cpuProfile, data.memProfile, data.routineProfile, data.blockProfile}
			}

			streamChan <- data

			profiling.SetStart(false)
		case <-tmChan:
			if !profiling.IsStarted() {
				break
			}

			logger.BkLog.Infof("Profiling reach time limit")

			go func() {
				select {
				case <-time.After(10 * time.Second):
					logger.BkLog.Infof("Timeout profiling timer")
				case profiling.stop <- make(chan []string):
				}
			}()
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

func initProfilingStream(profiling *ProfilingSetting, streamChan <-chan profileData) {
	logger.BkLog.Infof("Init profiling stream")
	for {
		select {
		case <-profiling.closed:
			return
		case data := <-streamChan:
			logger.BkLog.Infof("Streaming %v, %v, %v", data.cpuProfile, data.memProfile, data.routineProfile)
			now := time.Now()
			path := fmt.Sprintf("app-profiling/%s/%04d%02d%02d/", profiling.name, now.Year(), now.Month(), now.Day())
			if data.cpuProfile != "" {
				f, err := os.Open(data.cpuProfile)
				if err != nil {
					logger.BkLog.Errorw(fmt.Sprintf("Fail to open cpu profile: %v", err), "file", data.cpuProfile)
				} else {
					cpuFilename := fmt.Sprintf("cpu-%v-%s.pb.gz", now.Unix(), uuid.NewUUID().String())
					cpuUrl, err := utils.UploadS3(f, cpuFilename, path, "gzip")
					if err != nil {
						logger.BkLog.Errorw(fmt.Sprintf("Error when upload s3 file: %v", err))
					} else {
						logger.BkLog.Infof("Uploaded profile to s3: %v", cpuUrl)
					}
				}
			}

			if data.memProfile != "" {
				f, err := os.Open(data.memProfile)
				if err != nil {
					logger.BkLog.Errorw(fmt.Sprintf("Fail to open mem profile: %v", err), "file", data.memProfile)
				} else {
					memFilename := fmt.Sprintf("mem-%v-%s.pb.gz", now.Unix(), uuid.NewUUID().String())
					memUrl, err := utils.UploadS3(f, memFilename, path, "gzip")
					if err != nil {
						logger.BkLog.Errorw(fmt.Sprintf("Error when upload s3 file: %v", err))
					} else {
						logger.BkLog.Infof("Uploaded profile to s3: %v", memUrl)
					}
				}
			}

			if data.routineProfile != "" {
				f, err := os.Open(data.routineProfile)
				if err != nil {
					logger.BkLog.Errorw(fmt.Sprintf("Fail to open routine profile: %v", err), "file", data.routineProfile)
				} else {
					routineFilename := fmt.Sprintf("routine-%v-%s.pb.gz", now.Unix(), uuid.NewUUID().String())
					routineUrl, err := utils.UploadS3(f, routineFilename, path, "gzip")
					if err != nil {
						logger.BkLog.Errorw(fmt.Sprintf("Error when upload s3 file: %v", err))
					} else {
						logger.BkLog.Infof("Uploaded profile to s3: %v", routineUrl)
					}
				}
			}
		default:
			time.Sleep(1 * time.Second)
		}
	}
}
