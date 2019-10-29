package plugins

type Plugin interface {
	GetContent() <-chan []byte
	Write(data []byte) (n int, err error)
}
