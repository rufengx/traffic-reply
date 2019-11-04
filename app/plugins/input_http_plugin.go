package plugins

import (
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
	"time"
	"xtransform/app/common/httphandle"
	"xtransform/app/config"
)

// Build a http server, listen http request.
type HttpInputPlugin struct {
	httpServerConfig *config.HttpServerConfig

	msgLevel    int
	pluginName  string
	receiveChan chan *message
	httpServer  *http.Server

	IsDebug bool
}

func NewHttpInputPlugin(config *config.HttpServerConfig) (*HttpInputPlugin, error) {
	if nil == config {
		return nil, errors.New("params is empty")
	}
	plugin := new(HttpInputPlugin)
	plugin.msgLevel = msgLevelHttp
	plugin.pluginName = pluginNameInputHttp
	plugin.httpServerConfig = config
	plugin.receiveChan = make(chan *message, 4096)

	if err := plugin.listen(); nil != err {
		return nil, err
	}
	return plugin, nil
}

// Read request to data, transfer to next plugin.
func (plugin *HttpInputPlugin) GetMessage() <-chan *message {
	return plugin.receiveChan
}

func (plugin *HttpInputPlugin) Write(msg *message) (err error) {
	return nil
}

func (plugin *HttpInputPlugin) listen() error {
	config := plugin.httpServerConfig
	if plugin.IsDebug {
		log.Printf("Input-http-plugin config: %v \n", config)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", plugin.handler)

	httpServer := &http.Server{
		Addr:           config.Addr + ":" + strconv.Itoa(config.Port),
		Handler:        mux,
		ReadTimeout:    time.Duration(config.RTimeoutMs) * time.Millisecond,
		WriteTimeout:   time.Duration(config.WTimeoutMs) * time.Millisecond,
		IdleTimeout:    time.Duration(config.DTimeoutMs) * time.Millisecond,
		MaxHeaderBytes: config.MaxHeaderBytes,
	}
	plugin.httpServer = httpServer
	go func() {
		err := plugin.httpServer.ListenAndServe()
		panic(err)
	}()
	log.Printf("[Http-input-plugin] http server addr '%v'", httpServer.Addr)
	return nil
}

func (plugin *HttpInputPlugin) handler(w http.ResponseWriter, r *http.Request) {
	// Dump request body to receive queue.
	reqData, err := httputil.DumpRequest(r, true)
	if nil != err {
		httphandle.WriteJsonRaw(w, httphandle.CONFLICT, err.Error())
	} else {
		plugin.receiveChan <- &message{msgLevel: plugin.msgLevel, rawData: reqData, timestampNano: time.Now().UnixNano()}
		httphandle.WriteJson(w, httphandle.OK)
	}

	if plugin.IsDebug {
		log.Printf("Input-http-plugin receive request: \n %v \n", reqData)
	}
}

func (plugin *HttpInputPlugin) GetPluginName() string {
	return plugin.pluginName
}

func (plugin *HttpInputPlugin) Close() {
	plugin.httpServer.Close()
	close(plugin.receiveChan)
	log.Println("Close input-http-plugin finished.")
}
