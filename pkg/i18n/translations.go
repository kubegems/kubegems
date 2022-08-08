package i18n

import (
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

// Fprintf is like fmt.Fprintf, but using language-specific formatting.
func Fprintf(w io.Writer, key message.Reference, a ...interface{}) (n int, err error) {
	return p.printers[defaultLang].Fprintf(w, key, a...)
}

// Printf is like fmt.Printf, but using language-specific formatting.
func Printf(format string, a ...interface{}) {
	_, _ = p.printers[defaultLang].Printf(format, a...)
}

// Sprintf formats according to a format specifier and returns the resulting string.
func Sprintf(format string, a ...interface{}) string {
	return p.printers[defaultLang].Sprintf(format, a...)
}

// Sprint is like fmt.Sprint, but using language-specific formatting.
func Sprint(a ...interface{}) string {
	return p.printers[defaultLang].Sprint(a...)
}

func PrinterForLang(lang string) *message.Printer {
	langTag := message.MatchLanguage(lang)
	if printer, exist := p.printers[langTag]; exist {
		return printer
	}
	return p.printers[defaultLang]
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
