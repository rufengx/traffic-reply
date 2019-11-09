package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"xtransform/app/common/httpclient"
	"xtransform/app/config"
	"xtransform/app/listener"
	"xtransform/app/scheduler"
)

// TODO: add current version
func usage() {
	fmt.Println("Traffic Replay is a traffic replay software, it's main goal is redirect product traffic to dev or test environment. \nProject page: https://github.com/xy1884/traffic-reply \nAuthor: <Hang Dong> hangdongx@gmail.com")
	flag.PrintDefaults()
	os.Exit(2)
}

var IsDebug = flag.Bool("debug", false, "Debug mode, true is turn on debug mode, show all intercepted traffic.")
var inputHttpPort = flag.Int("input-http", -1, "Read http request in local http server, it's need to assign a port run http service.")
var outputHttpRedirectUrl = flag.String("output-http", "", "Forwards incoming requests to given http address. such as: --input-http 80 --output-http http://abc.com")

var inputRawOnLivePort = flag.Int("input-raw", -1, "Capture traffic in current active net interface card, listen special port traffic. such as: --input-raw 80 --output-http http://abc.com")

var outputTcpAddr = flag.String("output-tcp", "", "Forwards incoming packet to given tcp address. such as: --input-http 80 --output-tcp 127.0.0.1:8888")

func main() {
	flag.Usage = usage
	flag.Parse()

	*inputRawOnLivePort = 3800
	*outputHttpRedirectUrl = "http://127.0.0.1:3800/ab"
	*outputTcpAddr = "127.0.0.1:3800"

	fmt.Println("==============================")
	fmt.Println("input-http: ", *inputHttpPort)
	fmt.Println("input-raw: ", *inputRawOnLivePort)
	fmt.Println("output-http: ", *outputHttpRedirectUrl)
	fmt.Println("output-tcp: ", *outputTcpAddr)
	fmt.Println("==============================")

	// step 1: init app config
	appConfig := initAppConfig()

	// step 2: register plugin
	scheduler := scheduler.NewScheduler()
	err := scheduler.Init(appConfig)
	if nil != err {
		panic(err)
	}

	handleSignal(scheduler)
	log.Print("Traffic Reply exit. \n")
}

func initAppConfig() *config.AppConfig {
	appConfig := &config.AppConfig{}

	// case 1: http input plugin
	if *inputHttpPort > 0 {
		httpInputConfig := &config.HttpServerConfig{
			Port:           *inputHttpPort,
			RTimeoutMs:     1000,
			WTimeoutMs:     1000,
			DTimeoutMs:     1000,
			MaxHeaderBytes: 4096,
		}
		appConfig.HttpInputPluginConfig = httpInputConfig
	}

	// case 2: http output plugin
	if len(strings.TrimSpace(*outputHttpRedirectUrl)) > 0 {
		httpOutputConfig := &config.HttpOutputConfig{
			RedirectUrl: *outputHttpRedirectUrl,
			HttpRequestConfig: &httpclient.HttpRequestConfig{
				TimeoutMs: 1000,
			},
		}
		appConfig.HttpOutputPluginConfig = httpOutputConfig
	}

	// case 3: raw packet input plugin
	if *inputRawOnLivePort > 0 {
		rawInputPluginConfig := &config.RawInputConfig{
			RawSocketAddr: ":" + strconv.Itoa(*inputRawOnLivePort),
			PcapFilename:  "",
			DeviceName:    listener.AllDevice, // default capture all NICs traffic
			BpfFilter:     "tcp and dst port " + strconv.Itoa(*inputRawOnLivePort),
		}
		appConfig.RawInputPluginConfig = rawInputPluginConfig
	}

	// case 4: tcp output plugin
	if *outputTcpAddr != "" {
		appConfig.TcpOutputPluginConfig = *outputTcpAddr
	}

	return appConfig
}

func handleSignal(scheduler *scheduler.Scheduler) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	<-sigs
	scheduler.Close()
}
