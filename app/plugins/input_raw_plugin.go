package plugins

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
	"xtransform/app/config"
	"xtransform/app/listener"
)

// This private instance, main help process raw packet, see http stream run() function.
var rawInputPlugin *RawInputPlugin

// Read network interface card packet or raw socket packet.
type RawInputPlugin struct {
	rawSocketAddr string
	deviceName    string
	pcapFilename  string
	bpfFilter     string

	msgLevel    int
	pluginName  string
	receiveChan chan *message

	exit    bool
	IsDebug bool
}

func NewRawInputPlugin(config *config.RawInputConfig) (*RawInputPlugin, error) {
	if nil == config || (len(strings.TrimSpace(config.DeviceName)) == 0 &&
		len(strings.TrimSpace(config.PcapFilename)) == 0 &&
		len(strings.TrimSpace(config.RawSocketAddr)) == 0) {
		return nil, errors.New("invalid params")
	}

	plugin := &RawInputPlugin{
		rawSocketAddr: config.RawSocketAddr,
		deviceName:    config.DeviceName,
		pcapFilename:  config.PcapFilename,
		bpfFilter:     config.BpfFilter,

		msgLevel:    msgLevelPacket,
		pluginName:  pluginNameInputRaw,
		receiveChan: make(chan *message, 4096),
		IsDebug:     false,
	}

	if err := plugin.listen(); nil != err {
		return nil, err
	}
	rawInputPlugin = plugin // help process raw packet.
	return rawInputPlugin, nil
}

func (plugin *RawInputPlugin) listen() (err error) {
	// case 1: capture traffic on live
	var listenerOnLive *listener.Listener
	if len(strings.TrimSpace(plugin.deviceName)) != 0 {
		readMode := listener.ReadModeOnLive
		listenerOnLive, err = listener.NewListener(readMode, plugin.deviceName, "", plugin.bpfFilter)
		if nil != err {
			return err
		}

		if receivePacketChan, err := listenerOnLive.Listen(); nil == err {
			go plugin.processPacket(receivePacketChan)
		} else {
			return err
		}
	}

	// case 2: capture traffic on pcap file
	var listenerOnPcapFile *listener.Listener
	if len(strings.TrimSpace(plugin.pcapFilename)) != 0 {
		readMode := listener.ReadModeOnFile
		listenerOnPcapFile, err = listener.NewListener(readMode, "", plugin.pcapFilename, plugin.bpfFilter)
		if nil != err {
			return err
		}
		if receivePacketChan, err := listenerOnPcapFile.Listen(); nil == err {
			go plugin.processPacket(receivePacketChan)
		} else {
			return err
		}
	}

	// case 3: capture traffic on raw socket
	// TODO: capture raw socket

	return nil
}

func (plugin *RawInputPlugin) processPacket(receivePacketChan <-chan gopacket.Packet) {
	timeout := time.Duration(50) * time.Millisecond
	timer := time.NewTimer(timeout)

	streamFactory := &customStreamFactory{}
	streamPool := tcpassembly.NewStreamPool(streamFactory)
	assembler := tcpassembly.NewAssembler(streamPool)

	ticker := time.Tick(time.Minute)
	for {
		if plugin.exit {
			return
		}

		select {
		case packet := <-receivePacketChan:
			// case 1: tcp, http
			if packet.NetworkLayer() == nil || packet.TransportLayer() == nil || packet.TransportLayer().LayerType() != layers.LayerTypeTCP {
				log.Println("Unusable packet: ", packet)
				continue
			}
			tcp := packet.TransportLayer().(*layers.TCP)
			assembler.AssembleWithTimestamp(packet.NetworkLayer().NetworkFlow(), tcp, packet.Metadata().Timestamp)

			// case 2: udp
			// TODO:

			// case 3: socket
			// TODO:

		case <-ticker:
			// Every minute, flush connections that haven't seen activity in the past 2 minutes.
			assembler.FlushOlderThan(time.Now().Add(time.Minute * -2))
		default:
			<-timer.C
			timer.Reset(timeout)
		}
	}
}

func (plugin *RawInputPlugin) GetPluginName() string {
	return plugin.pluginName
}

func (plugin *RawInputPlugin) GetMessage() <-chan *message {
	return plugin.receiveChan
}

func (plugin *RawInputPlugin) Write(msg *message) (err error) {
	if nil == msg || msg.msgLevel != plugin.msgLevel {
		return errors.New("invalid params")
	}

	if plugin.exit {
		return errors.New("input-raw-plugin already closed")
	}
	//plugin.receiveChan <- msg
	return nil
}

// help func ===========================================================================================================

// parse packet, generate http request.
type customStreamFactory struct{}

// httpStream will handle the actual decoding of http requests.
type customStream struct {
	netFlow, tcpFlow gopacket.Flow
	reader           tcpreader.ReaderStream
}

func (factory *customStreamFactory) New(netFlow, tcpFlow gopacket.Flow) tcpassembly.Stream {
	reader := tcpreader.NewReaderStream()
	reader.LossErrors = false
	customStream := &customStream{
		netFlow: netFlow,
		tcpFlow: tcpFlow,
		reader:  reader,
	}
	go customStream.run() // start process http request
	return &customStream.reader
}

func (h *customStream) run() {
	for {
		// payload
		payload, err := ioutil.ReadAll(&h.reader)

		// build tcp message
		if (nil == err || err == io.EOF) && len(payload) > 0 {
			rawInputPlugin.receiveChan <- &message{msgLevel: msgLevelTcp, rawData: payload, timestampNano: time.Now().UnixNano()}
		}

		// build http request
		if request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(payload))); err == io.EOF {
			// We must read until we see an EOF... very important!
			return
		} else if err != nil {
			log.Println("Error reading stream", h.netFlow, h.tcpFlow, ":", err)
		} else {
			if nil != rawInputPlugin {
				if reqData, err := httputil.DumpRequest(request, true); nil == err {
					rawInputPlugin.receiveChan <- &message{msgLevel: msgLevelHttp, rawData: reqData, timestampNano: time.Now().UnixNano()}
				}
			}
		}
	}
}
