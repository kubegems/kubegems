package i18n

import (
	"errors"
	"io"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	p           *i18nPrinter
	defaultLang = language.English
)

type i18nPrinter struct {
	printers map[language.Tag]*message.Printer
}

var supported = language.NewMatcher([]language.Tag{
	language.AmericanEnglish,
	language.English,
	language.SimplifiedChinese,
	language.Chinese,
})

func InitWithLang(lang language.Tag) {
	tag, _, _ := supported.Match(lang)
	switch tag {
	case language.AmericanEnglish, language.English:
		initEnUS(lang)
	case language.SimplifiedChinese, language.Chinese:
		initZhCN(lang)
	default:
		initEnUS(lang)
	}
}

func langFromCtx(ctx *gin.Context) language.Tag {
	var lang language.Tag
	if ctx == nil {
		lang = defaultLang
		return lang
	}
	langs, _, err := language.ParseAcceptLanguage(ctx.GetHeader("Accept-Language"))
	lang, _, _ = supported.Match(langs...)
	if err != nil {
		lang = defaultLang
	} else {
		message.MatchLanguage()
	}
	return lang
}

func Fprintf(ctx *gin.Context, w io.Writer, key message.Reference, a ...interface{}) (n int, err error) {
	return p.printers[langFromCtx(ctx)].Fprintf(w, key, a...)
}

func Printf(ctx *gin.Context, format string, a ...interface{}) {
	_, _ = p.printers[langFromCtx(ctx)].Printf(format, a...)
}

func Sprintf(ctx *gin.Context, format string, a ...interface{}) string {
	return p.printers[langFromCtx(ctx)].Sprintf(format, a...)
}

func Sprint(ctx *gin.Context, a ...interface{}) string {
	return p.printers[langFromCtx(ctx)].Sprint(a...)
}

func Error(ctx *gin.Context, a ...interface{}) error {
	return errors.New(p.printers[langFromCtx(ctx)].Sprint(a...))
}

func Errorf(ctx *gin.Context, format string, a ...interface{}) error {
	return errors.New(p.printers[langFromCtx(ctx)].Sprintf(format, a...))
}

func init() {
	p = &i18nPrinter{}
	m := make(map[language.Tag]*message.Printer)
	langTags := []language.Tag{
		language.AmericanEnglish,
		language.English,
		language.SimplifiedChinese,
		language.Chinese,
	}
	for _, langTag := range langTags {
		switch langTag {
		case language.AmericanEnglish, language.English:
			initEnUS(langTag)
		case language.SimplifiedChinese, language.Chinese:
			initZhCN(langTag)
		}
		m[langTag] = message.NewPrinter(langTag)
	}
	p.printers = m
}
