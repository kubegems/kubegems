package i18n

import (
	"context"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

type CtxLang string

const LANG CtxLang = "lang"

func SetLang(c *gin.Context) {
	var lang language.Tag
	langs, _, err := language.ParseAcceptLanguage(c.GetHeader("Accept-Language"))
	lang, _, _ = supported.Match(langs...)
	if err != nil {
		lang = defaultLang
	}
	ctx := context.WithValue(c.Request.Context(), LANG, lang)
	c.Request = c.Request.WithContext(ctx)
}
