package handlers

import "kubegems.io/pkg/model/client"

type ResponseStruct struct {
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	ErrorData interface{} `json:"err"`
}

type PageData struct {
	Total       int64       `json:"total"`
	List        interface{} `json:"list"`
	CurrentPage int64       `json:"page"`
	CurrentSize int64       `json:"size"`
}

func PageList(l client.ObjectListIface, data interface{}) *PageData {
	page, size := l.GetPageSize()
	return &PageData{
		Total:       *l.GetTotal(),
		List:        data,
		CurrentPage: *page,
		CurrentSize: *size,
	}
}
