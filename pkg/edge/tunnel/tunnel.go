package tunnel

type Tunnel interface {
	Recv(*Packet) error
	Send(*Packet) error
}

type TunnelOptions struct {
	SendRouteChange bool
}

type ConnectedTunnel struct {
	Tunnel
	ID      string
	Options TunnelOptions
}
