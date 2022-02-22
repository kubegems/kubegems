package msgbus

type Options struct {
	Addr string `json:"addr" description:"msgbus host"`
}

func DefaultMsgbusOptions() *Options {
	return &Options{
		Addr: "http://gems-msgbus:8080",
	}
}
