package plugins

import (
	"bufio"
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"time"
	"xtransform/app/common/httpclient"
)

type HttpOutputConfig struct {
	Workers           int                           `yaml:"workers"`
	RedirectUrl       string                        `yaml:"redirect_url"`
	HttpRequestConfig *httpclient.HttpRequestConfig `yaml:"http_request_config"`
}

// Redirect received http request
type HttpOutputPlugin struct {
	redirectUrl *url.URL
	workers     int // it's define process worker process, default cores x 2
	config      *httpclient.HttpRequestConfig
	httpClient  *httpclient.HttpClient

	receiveChan chan []byte

	exit    bool
	IsDebug bool
}

func NewOutputHttpPlugin(config *HttpOutputConfig) (*HttpOutputPlugin, error) {
	if nil == config {
		return nil, errors.New("params is empty")
	}

	redirectUrl, err := url.Parse(config.RedirectUrl)
	if nil != err {
		return nil, err
	}

	// set default value
	if config.Workers == 0 {
		config.Workers = runtime.NumCPU() * 2
	}

	httpClient, err := httpclient.NewHttpClient(config.HttpRequestConfig)
	if nil != err {
		return nil, err
	}

	plugin := &HttpOutputPlugin{
		workers:     config.Workers,
		redirectUrl: redirectUrl,
		httpClient:  httpClient,
		receiveChan: make(chan []byte, 4096),
	}

	return plugin, nil
}

func (plugin *HttpOutputPlugin) Run() {
	for i := 0; i < plugin.workers; i++ {
		go plugin.worker()
	}
}

func (plugin *HttpOutputPlugin) worker() {
	timeout := time.Duration(50) * time.Millisecond
	timer := time.NewTimer(timeout)
	for {
		if plugin.exit {
			// clear
			for reqData := range plugin.receiveChan {
				plugin.send(reqData)
			}
			return
		}
		select {
		case reqData := <-plugin.receiveChan:
			plugin.send(reqData)
			// TODO: http stat
		default:
			<-timer.C
			timer.Reset(timeout)
		}
	}
}

func (plugin *HttpOutputPlugin) send(reqData []byte) (*http.Response, error) {
	if len(reqData) == 0 {
		return nil, errors.New("invalid params")
	}

	reader := bufio.NewReader(bytes.NewBuffer(reqData))
	req, err := http.ReadRequest(reader)
	if nil != err {
		return nil, err
	}
	// set redirect url
	req.URL = plugin.redirectUrl
	return plugin.httpClient.Do(req)
}

func (plugin *HttpOutputPlugin) Write(data []byte) error {
	if plugin.exit {
		return errors.New("output-http-plugin already closed")
	}
	plugin.receiveChan <- data
	return nil
}

func (plugin *HttpOutputPlugin) Close() {
	plugin.exit = true
	close(plugin.receiveChan)
	log.Println("Close output-http-plugin finished.")
}
