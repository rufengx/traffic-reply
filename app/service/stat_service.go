package service

// Statistics http request result
type HttpStatEntry struct {
	UUID string

	ReqtUrl   string
	ReqParams []byte

	ResStatusCode int
	ResBody       []byte

	RoundTripTimeNano int64
	StartTimeNano     int64
}
