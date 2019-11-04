package plugins

import (
	"bufio"
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"time"
	"xtransform/app/common/httpclient"
	"xtransform/app/config"
	"xtransform/app/service"
)

// Redirect received http request
type HttpOutputPlugin struct {
	msgLevel   int
	pluginName string

	redirectUrl *url.URL
	workers     int // it's define process worker process, default cores x 2
	config      *httpclient.HttpRequestConfig
	httpClient  *httpclient.HttpClient

	receiveChan chan *message

	exit    bool
	IsDebug bool
}

func NewOutputHttpPlugin(config *config.HttpOutputConfig) (*HttpOutputPlugin, error) {
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
		msgLevel:    msgLevelPacket + msgLevelTcp + msgLevelHttp,
		pluginName:  pluginNameOutputHttp,
		workers:     config.Workers,
		redirectUrl: redirectUrl,
		httpClient:  httpClient,
		receiveChan: make(chan *message, 4096),
	}

	go plugin.Run()
	return plugin, nil
}

func (plugin *HttpOutputPlugin) GetMessage() <-chan *message {
	return plugin.receiveChan
}

func (plugin *HttpOutputPlugin) Run() {
	for i := 0; i < plugin.workers; i++ {
		go plugin.productWorker() // http request producer
	}
}

func (plugin *HttpOutputPlugin) productWorker() {
	timeout := time.Duration(50) * time.Millisecond
	timer := time.NewTimer(timeout)

	for {
		if plugin.exit {
			return
		}
		select {
		case message := <-plugin.receiveChan:
			// case 1: parse http message
			if message.msgLevel == msgLevelHttp {
				plugin.send(message)
			}
		default:
			<-timer.C
			timer.Reset(timeout)
		}
	}
}

func (plugin *HttpOutputPlugin) send(msg *message) {
	// generate http request
	reader := bufio.NewReader(bytes.NewBuffer(msg.rawData))
	req, err := http.ReadRequest(reader)
	if nil != err {
		log.Println(err)
		return
	}

	// set redirect url
	req, err = http.NewRequest(req.Method, plugin.redirectUrl.String(), req.Body)

	statEntry := new(service.HttpStatEntry)
	startTimeNano := time.Now().UnixNano()
	res, err := plugin.httpClient.Do(req) // do http request
	endTimeNano := time.Now().UnixNano()
	if nil != err {
		statEntry.Err = err
	} else {
		// stat
		statEntry.ReqUrl = req.URL.String()
		statEntry.ResStatusCode = res.StatusCode
		resBody, _ := ioutil.ReadAll(res.Body)
		statEntry.ResBody = resBody
	}

	statEntry.StartTimeNano = startTimeNano
	statEntry.RoundTripTimeNano = endTimeNano - startTimeNano
	service.HttpStatService.Stat(statEntry)
}

func (plugin *HttpOutputPlugin) Write(msg *message) error {
	if plugin.exit {
		return errors.New("output-http-plugin already closed")
	}
	// use xor control access, refer to linux Access Control Lists.
	if (msg.msgLevel | plugin.msgLevel) != plugin.msgLevel {
		return errors.New("output-http-plugin message type not match")
	}
	plugin.receiveChan <- msg
	return nil
}

func (plugin *HttpOutputPlugin) GetPluginName() string {
	return plugin.pluginName
}

func (plugin *HttpOutputPlugin) Close() {
	plugin.exit = true
	close(plugin.receiveChan)
	log.Println("Close output-http-plugin finished.")
}
