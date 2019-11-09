package plugins

import (
	"errors"
	"log"
	"net"
	"strings"
	"time"
)

type TCPOutputPlugin struct {
	msgLevel   int
	pluginName string

	redirectAddr string
	workers      int // it's define process worker process, default cores x 2
	receiveChan  chan *message

	exit    bool
	IsDebug bool
}

func NewTCPOutputPlugin(addr string) (*TCPOutputPlugin, error) {
	if len(strings.TrimSpace(addr)) == 0 {
		return nil, errors.New("invalid params")
	}

	plugin := &TCPOutputPlugin{
		msgLevel:     msgLevelTcp,
		pluginName:   pluginNameOutputTcp,
		redirectAddr: addr,
		receiveChan:  make(chan *message, 4096),
	}

	go plugin.run()
	return plugin, nil
}

func (plugin *TCPOutputPlugin) run() {
	go plugin.productWorker()
}

func (plugin *TCPOutputPlugin) productWorker() {
	timeout := time.Duration(50) * time.Millisecond
	timer := time.NewTimer(timeout)

	addr, err := net.ResolveTCPAddr("tcp", plugin.redirectAddr)
	if nil != err {
		panic(err)
	}

	for {
		if plugin.exit {
			return
		}
		select {
		case message := <-plugin.receiveChan:
			// case 1: send tcp message
			if message.msgLevel == msgLevelTcp && len(message.rawData) > 0 {
				conn, err := net.DialTCP("tcp", nil, addr)
				if nil != err {
					log.Printf("[ouput-tcp-plugin] dial tcp fail, cause: %v", err.Error())
					plugin.receiveChan <- message
					break
				}
				defer conn.Close()

				if _, err := conn.Write(message.rawData); nil != err {
					plugin.receiveChan <- message
					go plugin.productWorker() // fail retry.
					break
				}
			}
		default:
			<-timer.C
			timer.Reset(timeout)
		}
	}
}

func (plugin *TCPOutputPlugin) GetPluginName() string {
	return plugin.pluginName
}
func (plugin *TCPOutputPlugin) GetMessage() <-chan *message {
	return plugin.receiveChan
}

func (plugin *TCPOutputPlugin) Write(msg *message) (err error) {
	if plugin.exit {
		return errors.New("output-tcp-plugin already closed")
	}
	// use xor control access, refer to linux Access Control Lists.
	if (msg.msgLevel | plugin.msgLevel) != plugin.msgLevel {
		return errors.New("output-tcp-plugin message type not match")
	}
	plugin.receiveChan <- msg
	return nil
}

func (plugin *TCPOutputPlugin) Close() {
	plugin.exit = true
	close(plugin.receiveChan)
	log.Println("Close output-tcp-plugin finished.")
}
