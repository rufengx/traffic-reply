package service

var TCPMessageService *tcpMessageService

type TCPMessage struct {
	ReqSeq uint32
	ReqAck uint32
	ResSeq uint32
	ResAck uint32

	tcpPackets []*TCPPacket
}

type tcpMessageService struct {
}
