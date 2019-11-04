package scheduler

import (
	"errors"
	"log"
	"sync"
	"time"
	"xtransform/app/config"
	"xtransform/app/plugins"
)

const Forever = -time.Millisecond * 10

// Register all plugins and write input-plugin traffic to output-plugin.
type Scheduler struct {
	mutex         sync.RWMutex
	inputPlugins  []plugins.Plugin
	outputPlugins []plugins.Plugin
	endpoints     map[int64]*Endpoint // timestamp nanosecond : endpoint
	exit          bool
}

// Input-plugin with Output-plugin relationship is N to M.
// Endpoint is a middle relationship, help maintain input and output.
type Endpoint struct {
	Input       plugins.Plugin
	Output      plugins.Plugin
	timeoutNano int64
}

func NewScheduler() *Scheduler {
	scheduler := &Scheduler{
		mutex:     sync.RWMutex{},
		endpoints: make(map[int64]*Endpoint),
		exit:      false,
	}
	return scheduler
}

func (s *Scheduler) Init(config *config.AppConfig) error {
	// case 1: init http input plugin
	if nil != config.HttpInputPluginConfig {
		httpInputPlugin, err := plugins.NewHttpInputPlugin(config.HttpInputPluginConfig)
		if nil != err {
			log.Println(err)
			return err
		}
		s.inputPlugins = append(s.inputPlugins, httpInputPlugin)
	}

	// case 2: init http output plugin
	if nil != config.HttpOutputPluginConfig {
		httpOutputPlugin, err := plugins.NewOutputHttpPlugin(config.HttpOutputPluginConfig)
		if nil != err {
			return err
		}
		s.outputPlugins = append(s.outputPlugins, httpOutputPlugin)
	}

	// case 3: init raw packet input plugin
	if nil != config.RawInputPluginConfig {
		rawInputPlugin, err := plugins.NewRawInputPlugin(config.RawInputPluginConfig)
		if nil != err {
			return err
		}
		s.inputPlugins = append(s.inputPlugins, rawInputPlugin)
	}

	log.Print("Scheduler init plugin finished, start register plugin ...")
	for _, in := range s.inputPlugins {
		for _, out := range s.outputPlugins {
			endpoint := &Endpoint{Input: in, Output: out, timeoutNano: time.Now().UnixNano()}
			s.RegisterEndpoint(endpoint)
			log.Printf("Register Endpoint (Input-Plugin: %s, Output-Plugin: %s) \n", in.GetPluginName(), out.GetPluginName())
		}
	}
	log.Print("Scheduler start service ...")
	return nil
}

func (s *Scheduler) RegisterEndpoint(endpoint *Endpoint) error {
	if nil == endpoint {
		return errors.New("invalid params")
	}
	if s.exit {
		return errors.New("Scheduler already closed")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.endpoints[endpoint.timeoutNano] = endpoint
	go s.transform(endpoint)
	return nil
}

func (s *Scheduler) transform(endpoint *Endpoint) {
	if nil == endpoint {
		return
	}
	// step 1: write input-plugin traffic to output-plugin
	timeout := time.Duration(50) * time.Millisecond
	timer := time.NewTimer(timeout)
	for {
		if s.exit {
			return
		}

		select {
		case data := <-endpoint.Input.GetMessage():
			// TODO: add middleware process
			endpoint.Output.Write(data)
		default:
			<-timer.C
			timer.Reset(timeout)
		}
	}
}

func (s *Scheduler) Close() {
	s.exit = true
}
