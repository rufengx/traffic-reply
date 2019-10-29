package httpclient

import (
	"encoding/base64"
	"errors"
	"golang.org/x/net/publicsuffix"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

const defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3865.120 Safari/537.36"

type HttpRequestConfig struct {
	TimeoutMs       int  `yaml:"timeout_ms"`
	MaxRetry        int  `yaml:"max_retry"`
	IsAllowRedirect bool `yaml:"is_allow_redirect"`
	MaxRedirects    int  `yaml:"max_redirects"`

	UA         string `yaml:"user_agent"`
	OriginHost string `yaml:"origin_host"`

	// http proxy, basic auth, See: https://en.wikipedia.org/wiki/Basic_access_authentication
	ProxyUrl      string `yaml:"proxy_url"`
	ProxyUsername string `yaml:"proxy_username"`
	ProxyPassword string `yaml:"proxy_password"`
}

type HttpClient struct {
	config     *HttpRequestConfig
	httpClient *http.Client
}

func NewHttpClient(config *HttpRequestConfig) (*HttpClient, error) {
	if nil == config {
		return nil, errors.New("param is empty")
	}

	// step 1: set proxy
	var transport *http.Transport
	proxyUrlStr := config.ProxyUrl

	if len(proxyUrlStr) > 0 {
		parseUrl, err := url.Parse(proxyUrlStr)
		if nil != err {
			log.Printf("Invalid proxy url: '%q' \n", proxyUrlStr)
			return nil, err
		}

		auth := config.ProxyUsername + ":" + config.ProxyPassword
		proxyAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))

		transport = &http.Transport{
			Proxy: http.ProxyURL(parseUrl),
			ProxyConnectHeader: http.Header{
				"Proxy-Authorization": {proxyAuth},
			},
		}
	}

	// step 2: set cookie
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if nil != err {
		return nil, err
	}

	// step 3: build http client
	httpClient := &http.Client{
		Timeout: time.Duration(config.TimeoutMs) * time.Millisecond,
		Jar:     jar,
	}
	if nil != transport {
		// if transport is nil, use default transport.
		httpClient.Transport = transport
	}

	if len(strings.TrimSpace(config.UA)) == 0 {
		config.UA = defaultUserAgent
	}
	return &HttpClient{
		config:     config,
		httpClient: httpClient,
	}, nil
}

func (hc *HttpClient) Do(request *http.Request) (*http.Response, error) {
	if nil == request {
		return nil, errors.New("param is empty")
	}

	request.Header.Set("User-Agent", hc.config.UA)
	request.Header.Set("Referer", hc.config.OriginHost)

	// TODO: retry
	res, err := hc.httpClient.Do(request)

	return res, err
}
