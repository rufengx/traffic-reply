package listener

import (
	"errors"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"net"
	"os"
	"path/filepath"
	"sync"
)

const (
	ReadModeOnLive = 0
	ReadModeOnFile = 1
)

const AllDevice = "all"

type Listener struct {
	mutex sync.RWMutex

	readMode     int
	deviceName   string
	pcapFilename string
	bpfFilter    string

	receiveChan chan gopacket.Packet
	exit        bool
}

func NewListener(readMode int, deviceName, filename string, bpfFilter string) (*Listener, error) {
	// validate file is exist
	if readMode == ReadModeOnFile {
		if "" == filename || len(filename) == 0 {
			return nil, errors.New("filename is empty")
		}
		filePathAbs, err := filepath.Abs(filename)
		if nil != err {
			return nil, err
		}
		fileInfo, err := os.Stat(filePathAbs)
		if nil != err {
			// file is not exist.
			return nil, err
		}
		if fileInfo.IsDir() {
			return nil, errors.New("file type error, file is dir")
		}
	}

	return &Listener{
		mutex:        sync.RWMutex{},
		readMode:     readMode,
		deviceName:   deviceName,
		pcapFilename: filename,
		bpfFilter:    bpfFilter,
		receiveChan:  make(chan gopacket.Packet),
		exit:         false,
	}, nil
}

func (l *Listener) Listen() (<-chan gopacket.Packet, error) {
	var err error
	if l.readMode == ReadModeOnLive {
		err = l.openLivePcap()
	} else {
		err = l.readPcapFile()
	}

	return l.receiveChan, err
}

func (l *Listener) openLivePcap() error {
	devices, err := pcap.FindAllDevs()
	if nil != err {
		panic(err)
	}

	for _, device := range devices {
		if device.Name != l.deviceName && l.deviceName != AllDevice {
			continue
		}

		go func(ifs pcap.Interface) {
			snapLen := int32(65535)
			if it, err := net.InterfaceByName(ifs.Name); err == nil {
				// auto-guess max length of packet to capture
				snapLen = int32(it.MTU + 68*2)
			}

			handle, err := pcap.OpenLive(ifs.Name, snapLen, true, pcap.BlockForever)
			if nil != err {
				panic(err)
			}
			defer handle.Close()

			if len(l.bpfFilter) > 0 {
				handle.SetBPFFilter(l.bpfFilter)
			}
			// Special case for tunnel interface
			// See: https://github.com/google/gopacket/issues/99
			var decoder gopacket.Decoder
			if 12 == handle.LinkType() {
				decoder = layers.LayerTypeIPv4
			} else {
				decoder = handle.LinkType()
			}

			packetSource := gopacket.NewPacketSource(handle, decoder)
			for packet := range packetSource.Packets() {
				if l.exit {
					return
				}
				l.receiveChan <- packet
			}
		}(device)
	}
	return nil
}

func (l *Listener) readPcapFile() error {
	handle, err := pcap.OpenOffline(l.pcapFilename)
	if nil != err {
		return err
	}
	defer handle.Close()

	if len(l.bpfFilter) > 0 {
		handle.SetBPFFilter(l.bpfFilter)
	}

	// Special case for tunnel interface
	// See: https://github.com/google/gopacket/issues/99
	var decoder gopacket.Decoder
	if 12 == handle.LinkType() {
		decoder = layers.LayerTypeIPv4
	} else {
		decoder = handle.LinkType()
	}

	packetSource := gopacket.NewPacketSource(handle, decoder)
	for packet := range packetSource.Packets() {
		if l.exit {
			break
		}
		l.receiveChan <- packet
	}
	return nil
}

func (l *Listener) Close() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.exit = true
}
