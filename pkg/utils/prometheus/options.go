package prometheus

type ExporterOptions struct {
	Listen string `json:"listen,omitempty" description:"listen address"`
}

func DefaultExporterOptions() *ExporterOptions {
	return &ExporterOptions{
		Listen: ":9100",
	}
}
