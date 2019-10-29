package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"xtransform/app/plugins"
)

type AppConfig struct {
	HttpInputPluginConfig  *HttpServerConfig         `yaml:"http_input_plugin_config"`
	HttpOutputPluginConfig *plugins.HttpOutputConfig `yaml:"http_output_plugin_config"`
}

type Option interface{}

type HttpServerConfig struct {
	Addr           string `yaml:"addr"`             // bind address
	Port           int    `yaml:"port"`             // bind port
	RTimeoutMs     int    `yaml:"request_timeout"`  // request timeout, in millisecond
	WTimeoutMs     int    `yaml:"response_timeout"` // response timeout, in millisecond
	DTimeoutMs     int    `yaml:"idle_timeout"`     // http connection idle timeout, in millisecond
	MaxHeaderBytes int    `yaml:"max_header_bytes"` // unit in byte

	// ssl support
	Ssl     bool   `yaml:"ssl"`
	SslCert string `yaml:"ssl_cert"`
	SslKey  string `yaml:"ssl_key"`

	// mesh support
	HTTP2    bool `yaml:"http2"`    // enable http2
	Healthz  bool `yaml:"healthz"`  // enable /-/healthz
	Throttle int  `yaml:"throttle"` // enable throttle if non negative, in time.Second/throttle ms
	Demotion int  `yaml:"demotion"` // enable demotion if non negative, max connections for listener
}

func InitConfig(filepath string) (*AppConfig, error) {
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	config := &AppConfig{}
	err = yaml.Unmarshal(file, config)
	if nil != err {
		return nil, err
	}
	return config, nil
}
