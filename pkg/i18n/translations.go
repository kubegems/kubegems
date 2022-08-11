package i18n

import (
	"context"
	"errors"
	"io"

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
	language.TraditionalChinese,
	language.Chinese,
})

func printerFromCtx(ctx context.Context) *message.Printer {
	var lang language.Tag
	if ctx == nil {
		lang = defaultLang
	} else {
		l := ctx.Value(LANG)
		if l == nil {
			lang = defaultLang
		} else {
			lang = l.(language.Tag)
		}
	}
	printer, exist := p.printers[lang]
	if exist {
		return printer
	}
	return p.printers[defaultLang]
}

func Fprintf(ctx context.Context, w io.Writer, key message.Reference, a ...interface{}) (n int, err error) {
	return printerFromCtx(ctx).Fprintf(w, key, a...)
}

func Printf(ctx context.Context, format string, a ...interface{}) {
	_, _ = printerFromCtx(ctx).Printf(format, a...)
}

func Sprintf(ctx context.Context, format string, a ...interface{}) string {
	return printerFromCtx(ctx).Sprintf(format, a...)
}

func Error(ctx context.Context, a ...interface{}) error {
	return errors.New(printerFromCtx(ctx).Sprint(a...))
}

func Errorf(ctx context.Context, format string, a ...interface{}) error {
	return errors.New(printerFromCtx(ctx).Sprintf(format, a...))
}

func init() {
	p = &i18nPrinter{}
	m := make(map[language.Tag]*message.Printer)
	langTags := []language.Tag{
		language.AmericanEnglish,
		language.English,
		language.SimplifiedChinese,
		language.TraditionalChinese,
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
