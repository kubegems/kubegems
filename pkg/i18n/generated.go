package i18n

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)
// initEnUS will init en_US support.
func initEnUS(tag language.Tag) {
	_ = message.SetString(tag, "login source not provide", "login source not provide")
	_ = message.SetString(tag, "message type %s is invalid", "message type %s is invalid")
	_ = message.SetString(tag, "source not exist", "source not exist")
}
// initZhCN will init zh_CN support.
func initZhCN(tag language.Tag) {
	_ = message.SetString(tag, "login source not provide", "登录源未提供")
	_ = message.SetString(tag, "message type %s is invalid", "消息类型 %s 无效")
	_ = message.SetString(tag, "source not exist", "源不存在")
}

