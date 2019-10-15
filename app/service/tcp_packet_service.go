package service

import (
	"encoding/binary"
	"github.com/google/gopacket"
)

var TCPPacketService *tcpPacketService

type TCPPacket struct {
	// tcp packet structure
	// See: https://en.wikipedia.org/wiki/Transmission_Control_Protocol

	// source ip + source port + destination ip + destination port + ack
	// generate unique packer id.
	TCPPacketID []byte

	// form network layer
	SourceIP      []byte
	DestinationIP []byte

	// form transport layer
	SourcePort      uint16
	DestinationPort uint16
	ReqSeq          uint32
	ReqAck          uint32
	DataOffset      uint8
	FIN             bool

	RawData []byte // full packet data
	TCPData []byte // transport layer data

	timestampNano int64 // nanosecond
}

type tcpPacketService struct {
}

func (tps *tcpPacketService) TCPPacketParse(packet gopacket.Packet) (*TCPPacket, error) {
	if nil == packet {
		return nil, nil
	}

	rawData := packet.Data()
	timestampNano := packet.Metadata().Timestamp.UnixNano()

	networkLayer := packet.NetworkLayer()
	if nil == networkLayer {
		return nil, nil
	}
	sourceIP, destinationIP := networkLayer.NetworkFlow().Endpoints()

	transportLayer := packet.TransportLayer()
	if nil == transportLayer {
		return nil, nil
	}
	sourcePort, destinationPort := transportLayer.TransportFlow().Endpoints()

	tcpData := transportLayer.LayerContents()

	sourceIPBytes := sourceIP.Raw()
	sourcePortBytes := sourcePort.Raw()
	destinationIPBytes := destinationIP.Raw()
	destinationPortBytes := destinationPort.Raw()
	reqSeqBytes := tcpData[4:8]

	reqSeq := binary.BigEndian.Uint32(reqSeqBytes)
	reqAck := binary.BigEndian.Uint32(tcpData[8:12])
	dataOffset := tcpData[12] & 0xF0 >> 4
	fin := tcpData[13]&0x01 != 0 // first bit is fin sign.

	tcpPacketID := []byte{}
	tcpPacketID = append(tcpPacketID, sourceIPBytes...)
	tcpPacketID = append(tcpPacketID, sourcePortBytes...)
	tcpPacketID = append(tcpPacketID, destinationIPBytes...)
	tcpPacketID = append(tcpPacketID, destinationPortBytes...)
	tcpPacketID = append(tcpPacketID, reqSeqBytes...)

	return &TCPPacket{
		TCPPacketID:     tcpPacketID,
		SourceIP:        sourceIP.Raw(),
		DestinationIP:   destinationIP.Raw(),
		SourcePort:      binary.BigEndian.Uint16(sourcePortBytes),
		DestinationPort: binary.BigEndian.Uint16(destinationPortBytes),
		ReqSeq:          reqSeq,
		ReqAck:          reqAck,
		DataOffset:      dataOffset,
		FIN:             fin,
		RawData:         rawData,
		TCPData:         tcpData,
		timestampNano:   timestampNano,
	}, nil
}
