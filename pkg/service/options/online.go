package options

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"gorm.io/gorm"
	"kubegems.io/pkg/log"
	clusterhandler "kubegems.io/pkg/service/handlers/cluster"
	microserviceoptions "kubegems.io/pkg/service/handlers/microservice/options"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/oauth"
	"kubegems.io/pkg/utils/prometheus"
)

type OptionsCheckable interface {
	CheckOptions() error
}

type OnlineOptions struct {
	Oauth        *oauth.Options                           `json:"oauth,omitempty"`
	Microservice *microserviceoptions.MicroserviceOptions `json:"microservice,omitempty"`
	Installer    *clusterhandler.InstallerOptions         `json:"installer,omitempty"`
	Monitor      *prometheus.MonitorOptions               `json:"monitor,omitempty"`

	m sync.Mutex
}

func (opts *OnlineOptions) Lock() {
	opts.m.Lock()
}

func (opts *OnlineOptions) UnLock() {
	opts.m.Unlock()
}

func NewOnlineOptions() *OnlineOptions {
	return &OnlineOptions{
		Oauth:        oauth.NewDefaultOptions(),
		Microservice: microserviceoptions.NewDefaultOptions(),
		Installer:    clusterhandler.DefaultInstallerOptions(),
		Monitor:      prometheus.DefaultMonitorOptions(),
	}
}

// 同步数据库配置到内存, 避免多副本时，一个修改了config，另一个不知道
func (opts *OnlineOptions) StartSync(db *gorm.DB, interval time.Duration) {
	for range time.NewTicker(interval).C {
		if err := opts.LoadFromDB(db); err != nil {
			log.Error(err, "load online options from db")
		}
		log.Debugf("load online options from db succeed")
	}
}

func (opts *OnlineOptions) convertToDBConfig() ([]models.OnlineConfig, error) {
	e := reflect.ValueOf(opts).Elem()
	cfgs := []models.OnlineConfig{}

	for i := 0; i < e.NumField(); i++ {
		if e.Type().Field(i).Name != "m" { // 'm' is the mutex name
			bts, err := json.Marshal(e.Field(i).Interface())
			if err != nil {
				return nil, fmt.Errorf("options %s, err: %w", e.Type().Field(i).Name, err)
			}
			cfgs = append(cfgs, models.OnlineConfig{
				Name:    e.Type().Field(i).Name,
				Content: bts,
			})
		}
	}
	return cfgs, nil
}

// 初始化到DB
func (opts *OnlineOptions) InitToDB(db *gorm.DB) error {
	cfgs, err := opts.convertToDBConfig()
	if err != nil {
		return err
	}

	for i := range cfgs {
		if err := db.FirstOrCreate(&cfgs[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

func (opts *OnlineOptions) SaveToDB(db *gorm.DB) error {
	cfgs, err := opts.convertToDBConfig()
	if err != nil {
		return err
	}

	return db.Save(&cfgs).Error
}

func (opts *OnlineOptions) LoadFromDB(db *gorm.DB) error {
	cfgs := []models.OnlineConfig{}
	if err := db.Find(&cfgs).Error; err != nil {
		return err
	}

	for _, cfg := range cfgs {
		if err := opts.CheckAndUpdateSipecifiedField(cfg); err != nil {
			return err
		}
	}
	return nil
}

func (opts *OnlineOptions) CheckAndUpdateSipecifiedField(cfg models.OnlineConfig) error {
	opts.Lock()
	defer opts.UnLock()

	e := reflect.ValueOf(opts).Elem()
	for i := 0; i < e.NumField(); i++ {
		if e.Type().Field(i).Name == cfg.Name {
			optIf := e.Field(i).Interface()
			// 先取出旧值
			oldContent, err := json.Marshal(optIf)
			if err != nil {
				return fmt.Errorf("marshal old cfg %s, err: %v", cfg.Name, err)
			}

			// 反序列化赋值
			if err := json.Unmarshal(cfg.Content, optIf); err != nil {
				return err
			}

			// 调用check
			checker, ok := optIf.(OptionsCheckable)
			if ok {
				// FIXME 如果有map，因map unmarshal机制
				// 若添加元素后调用check不通过，还原unmarshal时无法删除其中多的元素
				if err := checker.CheckOptions(); err != nil {
					// 新值check不通过，将旧值还原
					json.Unmarshal(oldContent, optIf)
					return fmt.Errorf("%s config err: %v", cfg.Name, err)
				}
			}

			// updated
			return nil
		}
	}
	return fmt.Errorf("unknown config name: %s", cfg.Name)
}
