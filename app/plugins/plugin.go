package plugins

// Message Level, help plugin process different type message.
const (
	msgLevelPacket = 1
	msgLevelTcp    = 2
	msgLevelSocket = 4
	msgLevelHttp   = 8
)

// Plugin Name, help sign all kinds of plugin.
const (
	pluginNameInputHttp  = "input-http-plugin"
	pluginNameOutputHttp = "output-http-plugin"

	pluginNameInputRaw  = "input-raw-plugin"
	pluginNameOutputRaw = "output-raw-plugin"
)

// Define plugin message process range, you can to set multi different range. this section refer to linux Access Control Lists.
type message struct {
	msgLevel      int // http : 8, socket : 4, tcp : 2, packet = 1
	rawData       []byte
	data          []byte
	timestampNano int64
}

type Plugin interface {
	GetPluginName() string
	GetMessage() <-chan *message
	Write(msg *message) (err error)
}
