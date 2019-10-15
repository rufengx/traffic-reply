package service

var ScheduleCenter *scheduleCenter

type scheduleCenter struct {
}

func (sc *scheduleCenter) listener() {
	// 1. 监听网卡，获取数据包。或者从文件中读取 pcap
	// 2. 根据捕获的包，组成 message
	// 3. 根据设置，将包转发到不同的地方，接口，数据库，文件
}

func (sc *scheduleCenter) dispatch() {

}
