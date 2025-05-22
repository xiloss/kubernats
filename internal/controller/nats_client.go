package controller

import "github.com/nats-io/nats.go"

type Connection interface {
	Status() nats.Status
	Close()
}

type Client interface {
	Connect(url string) (Connection, error)
	JetStream(nc Connection) (JetStreamContext, error)
}

type JetStreamContext interface {
	StreamInfo(name string) (*nats.StreamInfo, error)
	AddStream(cfg *nats.StreamConfig) (*nats.StreamInfo, error)
}

type ClientImpl struct{}

func (c *ClientImpl) Connect(url string) (*nats.Conn, error) {
	return nats.Connect(url)
}

func (c *ClientImpl) JetStream(nc *nats.Conn) (nats.JetStreamContext, error) {
	return nc.JetStream()
}

func (c *ClientImpl) AddStream(js nats.JetStreamContext, cfg *nats.StreamConfig) (*nats.StreamInfo, error) {
	return js.AddStream(cfg)
}
