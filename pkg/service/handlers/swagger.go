/*
some struct for swagger used
*/
package handlers

type ResponseStruct struct {
	Message   string
	Data      interface{}
	ErrorData interface{}
}

type PageData struct {
	Total       int64
	List        interface{}
	CurrentPage int64
	CurrentSize int64
}
