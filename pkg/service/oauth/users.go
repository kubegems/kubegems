package oauth

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/go-resty/resty/v2"
	"kubegems.io/pkg/log"
)

// 竹云 用户数据同步
// TODO: 完成写入用户数据逻辑(BamBooMessage中的data)

const (
	MethodLogin    = "login"
	MethodLogout   = "logout"
	MethodPullTask = "pullTask"
)

type BamBooException struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

type BamBooLoginRequest struct {
	SystemCode     string `json:"systemCode"`
	IntegrationKey string `json:"integrationKey"`
	Force          bool   `json:"force"`
	TimeStamp      int64  `json:"timestamp"`
}

type BamBooLoginOutRequest struct {
	TimeStamp int64  `json:"timestamp"`
	TokenID   string `json:"tokenId"`
}

type BamBooMessage struct {
	Success   bool  `json:"success"`
	TimeStamp int64 `json:"timestamp"`

	Message   string          `json:"message"`
	Exception BamBooException `json:"exception"`

	TokenID    string `json:"tokenId"`
	SystemCode string `json:"systemCode"`
	SystemName string `json:"systemName"`

	Data map[string]interface{} `json:"data"`
}

// 竹云用户同步工具
type BamBooUserSyncTool struct {
	BambooOptions
	token string
}

type BambooOptions struct {
	Host           string
	SystemCode     string
	IntegrationKey string
}

func NewDefaultBambooOptions() *BambooOptions {
	return &BambooOptions{
		Host:           "",
		SystemCode:     "",
		IntegrationKey: "",
	}
}

var bambooOptions *BambooOptions

func InitBamBooSyncTool(opts *BambooOptions) {
	bambooOptions = opts
}

func NewBamBooSyncTool() *BamBooUserSyncTool {
	host := bambooOptions.Host
	systemCode := bambooOptions.SystemCode
	integrationKey := bambooOptions.IntegrationKey
	if len(host) == 0 {
		log.Warnf("bamboo cloud host is empty")
	}
	if len(systemCode) == 0 {
		log.Warnf("bamboo cloud systemcode is empty")
	}
	if len(integrationKey) == 0 {
		log.Warnf("bamboo cloud integrationKey is empty")
	}
	return &BamBooUserSyncTool{BambooOptions: *bambooOptions}
}

func (s *BamBooUserSyncTool) geturi(kind string) string {
	var args string
	switch kind {
	case MethodLogin:
		argBytes, _ := json.Marshal(BamBooLoginRequest{SystemCode: s.SystemCode, IntegrationKey: s.IntegrationKey, Force: true, TimeStamp: time.Now().UnixNano()})
		args = string(argBytes)
	case MethodLogout:
		argBytes, _ := json.Marshal(BamBooLoginOutRequest{TokenID: s.token, TimeStamp: time.Now().UnixNano()})
		args = string(argBytes)
	case MethodPullTask:
		argBytes, _ := json.Marshal(BamBooLoginOutRequest{TokenID: s.token, TimeStamp: time.Now().UnixNano()})
		args = string(argBytes)
	default:
		args = ""
	}
	values := url.Values{}
	values.Add("method", kind)
	values.Add("request", args)
	urlobj := url.URL{Host: s.Host, Path: "/bim-server/integration/api.json", RawQuery: values.Encode()}
	return urlobj.String()
}

func (s *BamBooUserSyncTool) DoRequest(uri string, result *BamBooMessage) error {
	client := resty.New()
	_, err := client.R().SetResult(&result).SetError(&result).Get(uri)
	return err
}

func (s *BamBooUserSyncTool) Login() error {
	uri := s.geturi(MethodLogin)
	ret := &BamBooMessage{}
	if e := s.DoRequest(uri, ret); e != nil {
		return e
	}
	s.token = ret.TokenID
	return nil
}

func (s *BamBooUserSyncTool) Logout() error {
	uri := s.geturi(MethodLogout)
	ret := &BamBooMessage{}
	if e := s.DoRequest(uri, ret); e != nil {
		return e
	}
	if ret.Success {
		return nil
	}
	return fmt.Errorf("logout failed %v", ret.Message)
}

func (s *BamBooUserSyncTool) Do(syncData *BamBooMessage) error {
	uri := s.geturi(MethodPullTask)
	ret := &BamBooMessage{}
	if e := s.DoRequest(uri, ret); e != nil {
		return e
	}
	if !ret.Success {
		return fmt.Errorf("failed to sync data %v", ret.Message)
	}
	// TODO: 解析数据，完成数据同步
	// data := ret.Data
	return nil
}
