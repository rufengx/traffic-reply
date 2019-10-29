package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
	"sync"
	"time"
	"xtransform/app/common/httpclient"
)

func main() {
	//service, err := service.NewHttpAssemblyService("lo0", "", "tcp and dst port 3800", "http://localhost:3800/api/v3/ab")
	//if nil != err {
	//	panic(err)
	//}
	//
	//go func() {
	//	time.Sleep(10 * time.Second)
	//	service.Stop()
	//}()
	//
	//service.Run()

	//client()

	exit()
}

/**

1. 获取所有的配置，命令行
2. 初始化各个插件
3. 对接各个插件，输入拷贝到输出
4. 开启统计

*/

func client() {

	reqConfig := &httpclient.HttpRequestConfig{
		TimeoutMs: 6000,
	}
	httpclient, err := httpclient.NewHttpClient(reqConfig)

	if nil != err {
		panic(err)
	}

	req, err := http.NewRequest(http.MethodGet, "https://www.baidu.com/@nitishkr88/http-retries-in-go-e622e51d249f", nil)
	if nil != err {
		panic(err)
	}

	res, err := httpclient.Do(req)
	if nil != err {
		panic(err)
	}

	fmt.Println(res)
}

func exit() {
	wg := sync.WaitGroup{}

	exitChan := make(chan struct{})
	exit := false

	for i := 0; i <= 3; i++ {
		go func(i int) {
			timeout := time.Duration(50) * time.Millisecond
			timer := time.NewTimer(timeout)
			for {
				if exit {
					fmt.Println(i)
					return
				}
				select {
				case <-exitChan:
					wg.Done()
				default:
					<-timer.C
					timer.Reset(timeout)
				}
			}
		}(i)
		wg.Add(1)
	}
	go func() {
		exitChan <- struct{}{}
		exit = true
		wg.Done()
	}()
	wg.Add(1)
	wg.Wait()

}

func dump() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dump, err := httputil.DumpRequest(r, true)
		if err != nil {
			http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "dump: %q", dump)

		buff := bytes.NewBuffer(dump)
		reader := bufio.NewReader(buff)

		//url, err := url.Parse("http://www.baidu.com")

		req, err := http.ReadRequest(reader)
		fmt.Println(req)

		//req, err := http.NewRequest("POST", "http://www.baidu.com", reader)
		b, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("=========")
		fmt.Println(string(b))
		fmt.Println("=========")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("new: ", resp)
	}))
	defer ts.Close()

	const body = "Go is a general-purpose language designed with systems programming in mind."
	req, err := http.NewRequest("POST", ts.URL, strings.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	req.Host = "www.example.org"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s", b)

}
