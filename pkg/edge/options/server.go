package options

type ServerOptions struct {
	Listen string `json:"listen,omitempty"`
	TLS    *TLS   `json:"tLs,omitempty"`
}

func NewDefaultServer() *ServerOptions {
	return &ServerOptions{
		Listen: ":8088",
		TLS:    NewDefaultTLS(),
	}
}
