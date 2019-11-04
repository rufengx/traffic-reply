package service

import "fmt"

var HttpStatService *httpStatService

// Statistics http request result
type HttpStatEntry struct {
	ReqUrl string

	ResStatusCode int
	ResBody       []byte

	Err error

	RoundTripTimeNano int64
	StartTimeNano     int64
}

type httpStatService struct {
}

func (s *httpStatService) Stat(entry *HttpStatEntry) {
	// TODO
	c := fmt.Sprintf("request url: %v, response status code: %v, err: %v \n", entry.ReqUrl, entry.ResStatusCode, entry.Err)
	fmt.Println(c)
}
