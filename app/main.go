package main

import (
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"strings"
	"time"
)

type Config struct {
	// read config
	ReadForm     string // eth or file
	ReadFormEth  string // eth name
	ReadFormFile string // filename

	// write config
	WriteTo           string // http, file
	WriteToHttp       string // url
	WriteToHttpMethod string // http method: get/post/put/delete etc...
	WriteToFile       string // filename

	// filter config
	AllowDomains []string
	AllowIPs     []string
	AllowPorts   []int
}

const (
	ReadFormEth  = "eth"
	ReadFormFile = "file"
)

var config *Config

func main() {
	// 功能：
	// 指定网卡 en0

	// WIFI 探测

	// 辅助功能
	// 1. 需要当前环境的root权限
	// 2. 列出当前环境的所有网卡，供选择

	// 监听来源（目标网卡，已转储文件）
	// 过滤（IP, Domain, Port）
	// 转发方向（重新生成请求，转储成文件）

	config = &Config{
		ReadForm:    "eth",
		ReadFormEth: "en0",

		AllowIPs: []string{"111.231.103.186"},
	}
	readForm(config)
}

// read func ===========================================================================================================
func readForm(config *Config) error {
	if ReadFormEth == config.ReadForm {
		readFormEth(config.ReadFormEth)
	} else if ReadFormFile == config.ReadForm {
		readFromFile(config.ReadFormFile)
	} else {
		return errors.New("not support current read type: " + config.ReadForm)
	}
	return nil
}

func readFormEth(eth string) error {
	eth, isok := findEth(eth)
	if !isok {
		return errors.New("not found eth: " + eth)
	}

	readLivePackets(eth, 1600, false)
	return nil
}

func readFromFile(filename string) {
	// TODO: ...
}

// write func ==========================================================================================================
func writeTo(target string) {

}

func writeToHttp(url string, method string) {

}

func writeToFile(filename string) {

}

// packet func =========================================================================================================
func findEth(deviceName string) (string, bool) {
	ifs, err := pcap.FindAllDevs()
	if nil != err {
		panic(err)
	}

	if "" == deviceName {
		// if device is null than default select first eth.
		return ifs[0].Name, true
	}

	// check eth is exist.
	for _, device := range ifs {
		if strings.EqualFold(deviceName, device.Name) {
			return deviceName, true
		}
	}
	return "", false
}

func readLivePackets(eth string, snaplen int32, promisc bool) {
	handle, err := pcap.OpenLive(eth, snaplen, promisc, pcap.BlockForever)
	if nil != err {
		panic(err)
	}

	handle.SetBPFFilter(genFilter())

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packetChan := packetSource.Packets()

	duration := time.Microsecond * 50
	timer := time.NewTimer(duration)
	for {
		timer.Reset(duration)
		select {
		case packet := <-packetChan:
			packetProcess(packet)
		default:
			<-timer.C
		}
	}
}

func packetProcess(packet gopacket.Packet) {

	// fmt.Println(string(packet.Layers()[3].LayerContents()))

	fmt.Println(packet.Metadata().CaptureInfo)

	isok := filterIP(packet, config.AllowIPs)
	if isok {
		fmt.Println(packet.NetworkLayer().NetworkFlow().String())
		fmt.Println(packet.NetworkLayer().NetworkFlow())
	}

}

// filter func =========================================================================================================
func genFilter() string {
	// TODO: ...
	return ""
}

func filterDomain(packet gopacket.Packet, domains []string) bool {
	if nil == packet || len(domains) == 0 {
		return false
	}
	return false
}

func filterIP(packet gopacket.Packet, ips []string) bool {
	if nil == packet || len(ips) == 0 {
		// if ips is null, allow all packet though.
		return true
	}

	dst := packet.NetworkLayer().NetworkFlow().Dst().String()
	for _, ip := range ips {
		if strings.EqualFold(ip, dst) {
			return true
		}
	}
	return false
}

func filterPort(packet gopacket.Packet, ports []int) bool {
	if nil == packet || len(ports) == 0 {
		return false
	}

	return false
}
