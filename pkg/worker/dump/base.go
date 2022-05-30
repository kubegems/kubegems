package dump

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/prometheus/common/model"
	"github.com/robfig/cron/v3"
	"github.com/spf13/pflag"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils"
)

type DumpOptions struct {
	Dir               string `yaml:"dir"`
	ExecCron          string `yaml:"execCron"`
	DataStoreDuration string `yaml:"dataStoreDuration"`
}

func (o *DumpOptions) RegistFlags(prefix string, fs *pflag.FlagSet) {
	fs.StringVar(&o.Dir, utils.JoinFlagName(prefix, "dir"), o.Dir, "mysql dump file dir")
	fs.StringVar(&o.ExecCron, utils.JoinFlagName(prefix, "execCron"), o.ExecCron, "mysql dump exec cron expression, please refer https://en.wikipedia.org/wiki/Cron")
	fs.StringVar(&o.DataStoreDuration, utils.JoinFlagName(prefix, "dataStoreDuration"), o.DataStoreDuration, "date store duration, eg. 7d, 30d")
}

func NewDefaultDumpOptions() *DumpOptions {
	return &DumpOptions{
		Dir:               "data/dump", // 默认当前路径下的dump
		ExecCron:          "@daily",
		DataStoreDuration: "30d",
	}
}

func (d *Dump) Start() {
	cron := cron.New()
	dur, err := model.ParseDuration(d.Options.DataStoreDuration)
	if err != nil {
		log.Fatalf(err.Error())
	}

	if _, err := cron.AddFunc(d.Options.ExecCron, func() {
		d.ExportMessages(d.Options.Dir, time.Duration(dur))
	}); err != nil {
		log.Fatalf(err.Error())
	}
	if _, err := cron.AddFunc(d.Options.ExecCron, func() {
		d.ExportAuditlogs(d.Options.Dir, time.Duration(dur))
	}); err != nil {
		log.Fatalf(err.Error())
	}
	cron.Start()
}

func getDumpFile(dirpath, module string, year int, mon time.Month) (*os.File, error) {
	// 使用截止当月作为文件名，保证同一月的数据写入同一个文件
	filename := path.Join(dirpath, fmt.Sprintf("%s.%d-%d.dump.csv", module, year, mon))
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	// 写入utf8 bom，防止csv打开乱码
	bomUtf8 := []byte{0xEF, 0xBB, 0xBF}
	if _, err := file.Write(bomUtf8); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}
