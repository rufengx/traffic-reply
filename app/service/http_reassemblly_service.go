package service

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
	"xtransform/app/listener"
)

// it's help http stream process request redirect.
var httpAssemblySrc *httpAssemblyService

type httpAssemblyService struct {
	deviceName   string
	pcapFilename string
	bpfFilter    string
	redirectUrl  *url.URL
	IsDebug      bool
	exitChan     chan bool

	// stat request info, contain success, fail and other.
	counter map[string]int64
}

func NewHttpAssemblyService(deviceName, pcapFilename, bpfFileter string, redirectUrl string) (*httpAssemblyService, error) {
	if (len(bpfFileter) == 0 || len(redirectUrl) == 0) && len(deviceName) == 0 && len(pcapFilename) == 0 {
		return nil, errors.New("params is empty")
	}

	// url parse
	url, err := url.Parse(redirectUrl)
	if nil != err {
		return nil, err
	}

	service := &httpAssemblyService{
		deviceName:   deviceName,
		pcapFilename: pcapFilename,
		bpfFilter:    bpfFileter,
		redirectUrl:  url,
		IsDebug:      false,
		exitChan:     make(chan bool),
		counter:      make(map[string]int64),
	}
	httpAssemblySrc = service
	return httpAssemblySrc, nil
}

func (h *httpAssemblyService) Run() {
	defer func() {
		if err := recover(); nil != err {
			log.Println("HttpAssemblyService runtime error. cause: ", err)
		}
	}()

	log.Println(fmt.Sprintf("HttpAssemblyService start work. deviceName: '%s', pcapFilename: '%s', bpfFilter: '%s', redirectUrl: '%s'", h.deviceName, h.pcapFilename, h.bpfFilter, h.redirectUrl.String()))
	listener, err := listener.NewListener(listener.ReadModeOnLive, h.deviceName, h.pcapFilename, h.bpfFilter)
	if nil != err {
		panic(err)
	}

	packetsChan, err := listener.Listen()
	if nil != err {
		panic(err)
	}
	defer listener.Close()

	go h.reassembly(packetsChan)

	// wait exit.
	<-h.exitChan

	// display stat info
	h.display()
}

func (h *httpAssemblyService) Stop() {
	h.exitChan <- true
}

func (h *httpAssemblyService) reassembly(packetsChan <-chan gopacket.Packet) {
	streamFactory := &httpStreamFactory{}
	streamPool := tcpassembly.NewStreamPool(streamFactory)
	assembler := tcpassembly.NewAssembler(streamPool)
	ticker := time.Tick(time.Minute)
	for {
		select {
		case packet := <-packetsChan:
			// A nil packet indicates the end of a pcap file.
			if packet == nil {
				return
			}
			if h.IsDebug {
				log.Println(packet)
			}
			if packet.NetworkLayer() == nil || packet.TransportLayer() == nil || packet.TransportLayer().LayerType() != layers.LayerTypeTCP {
				log.Println("Unusable packet: ", packet)
				continue
			}
			tcp := packet.TransportLayer().(*layers.TCP)
			assembler.AssembleWithTimestamp(packet.NetworkLayer().NetworkFlow(), tcp, packet.Metadata().Timestamp)

		case <-ticker:
			// Every minute, flush connections that haven't seen activity in the past 2 minutes.
			assembler.FlushOlderThan(time.Now().Add(time.Minute * -2))

		case <-h.exitChan:
			return
		}
	}
	h.exitChan <- true
}

func (h *httpAssemblyService) redirect(req *http.Request) {
	if nil == req {
		return
	}

	request, err := http.NewRequest(req.Method, h.redirectUrl.String(), req.Body)
	if nil != err {
		log.Println("Http Handler new request fail, cause: ", err.Error())
	}
	res, err := http.DefaultClient.Do(request)
	if nil != err {
		count := h.counter[err.Error()]
		h.counter[err.Error()] = count + 1
		log.Println("Http Handler do redirect fail, cause: ", err.Error())
	} else {
		count := h.counter[res.Status]
		h.counter[res.Status] = count + 1
		if h.IsDebug {
			fmt.Println(fmt.Sprintf("redirect to '%s', result: %s.", h.redirectUrl.String(), res.Status))
		}
	}
}

func (h *httpAssemblyService) display() {
	fmt.Println("================ http redirect stat info ================")
	for key, value := range h.counter {
		fmt.Println(fmt.Sprintf("'%s' --> %d ", key, value))
	}
	fmt.Println("=========================================================")
}

type httpStreamFactory struct{}

// httpStream will handle the actual decoding of http requests.
type httpStream struct {
	netFlow, tcpFlow gopacket.Flow
	reader           tcpreader.ReaderStream
}

func (factory *httpStreamFactory) New(netFlow, tcpFlow gopacket.Flow) tcpassembly.Stream {
	reader := tcpreader.NewReaderStream()
	reader.LossErrors = false
	httpStream := &httpStream{
		netFlow: netFlow,
		tcpFlow: tcpFlow,
		reader:  reader,
	}
	go httpStream.run() // start process http request
	return &httpStream.reader
}

func (h *httpStream) run() {
	buf := bufio.NewReader(&h.reader)
	for {
		if req, err := http.ReadRequest(buf); err == io.EOF {
			// We must read until we see an EOF... very important!
			return
		} else if err != nil {
			log.Println("Error reading stream", h.netFlow, h.tcpFlow, ":", err)
		} else {
			// redirect
			httpAssemblySrc.redirect(req)
		}
	}
}
