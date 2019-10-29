package plugins

import (
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"time"
	"xtransform/app/config"
)

// Build a http server, listen http request.
type HttpInputPlugin struct {
	httpServerConfig *config.HttpServerConfig

	receiveChan chan []byte
	httpServer  *http.Server

	IsDebug bool
}

func NewHttpInputPlugin(config *config.HttpServerConfig) (*HttpInputPlugin, error) {
	if nil == config {
		return nil, errors.New("params is empty")
	}

	plugin := &HttpInputPlugin{
		httpServerConfig: config,
		receiveChan:      make(chan []byte, 4096),
	}
	if err := plugin.listen(); nil != err {
		return nil, err
	}
	return plugin, nil
}

// Read request to data, transfer to next plugin.
func (plugin *HttpInputPlugin) GetContent() <-chan []byte {
	return plugin.receiveChan
}

func (plugin *HttpInputPlugin) listen() error {
	config := plugin.httpServerConfig
	if plugin.IsDebug {
		log.Printf("Input-http-plugin config: %v \n", config)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", plugin.handler)

	httpServer := &http.Server{
		Addr:           config.Addr,
		Handler:        mux,
		ReadTimeout:    time.Duration(config.RTimeoutMs) * time.Millisecond,
		WriteTimeout:   time.Duration(config.WTimeoutMs) * time.Millisecond,
		IdleTimeout:    time.Duration(config.DTimeoutMs) * time.Millisecond,
		MaxHeaderBytes: config.MaxHeaderBytes,
	}
	plugin.httpServer = httpServer
	err := plugin.httpServer.ListenAndServe()
	return err
}

func (plugin *HttpInputPlugin) handler(w http.ResponseWriter, r *http.Request) {
	// Dump request body to receive queue.
	req, err := httputil.DumpRequest(r, true)
	http.Error(w, err.Error(), 200)
	plugin.receiveChan <- req

	if plugin.IsDebug {
		log.Printf("Input-http-plugin receive request: \n %v \n", req)
	}
}

func (plugin *HttpInputPlugin) Close() {
	plugin.httpServer.Close()
	close(plugin.receiveChan)
	log.Println("Close input-http-plugin finished.")
}
