package main

import (
	"time"
	"xtransform/app/service"
)

func main() {
	service, err := service.NewHttpAssemblyService("lo0", "", "tcp and dst port 3800", "http://localhost:3800/api/v3/ab")
	if nil != err {
		panic(err)
	}

	go func() {
		time.Sleep(10 * time.Second)
		service.Stop()
	}()

	service.Run()

}
