package service

import (
	"errors"
	"sync"
	"time"
	"xtransform/app/plugins"
)

const Forever = -time.Millisecond * 10

// Register all plugins and write input-plugin traffic to output-plugin.
type scheduler struct {
	mutex     sync.RWMutex
	endpoints map[string]*Endpoint // unique_key : endpoint
	exit      bool
}

// Input-plugin with Output-plugin relationship is N to M.
// Endpoint is a middle relationship, help maintain input and output.
type Endpoint struct {
	UUID      string
	Input     plugins.Plugin
	Output    plugins.Plugin
	timeoutMs int64
}

func NewScheduler() (*scheduler, error) {
	scheduler := &scheduler{
		mutex:     sync.RWMutex{},
		endpoints: make(map[string]*Endpoint),
		exit:      false,
	}
	return scheduler, nil
}

func (s *scheduler) Register(endpoint *Endpoint) error {
	if nil == endpoint {
		return errors.New("invalid params")
	}
	if s.exit {
		return errors.New("scheduler already closed")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.endpoints[endpoint.UUID] = endpoint
	go s.transform(endpoint)
	return nil
}

func (s *scheduler) transform(endpoint *Endpoint) {
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
		case data := <-endpoint.Input.GetContent():
			endpoint.Output.Write(data)
		default:
			<-timer.C
			timer.Reset(timeout)
		}
	}

	for data := range endpoint.Input.GetContent() {
		// TODO: add middleware process
		endpoint.Output.Write(data)
	}
}

func (s *scheduler) Close() {
	s.exit = true
}
