// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package i18n

import (
	"context"
	"errors"
	"io"

	"github.com/Xuanwo/go-locale"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	p             *i18nPrinter
	defaultLang   language.Tag
	supportedTags = []language.Tag{
		language.AmericanEnglish,
		language.English,
		language.Japanese,
		language.SimplifiedChinese,
		language.TraditionalChinese,
		language.Chinese,
	}
)

var matcher = language.NewMatcher(supportedTags)

type i18nPrinter struct {
	printers map[language.Tag]*message.Printer
}

func printerFromCtx(ctx context.Context) *message.Printer {
	var lang language.Tag
	if ctx == nil {
		return p.printers[defaultLang]
	}
	l := ctx.Value(LANG)
	if l == nil {
		lang = defaultLang
	} else {
		lang = l.(language.Tag)
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
	tag, err := locale.Detect()
	if err != nil {
		tag = language.SimplifiedChinese
	}
	matchedLang, _, _ := matcher.Match(tag)
	defaultLang = matchedLang.Parent()
	p = &i18nPrinter{}
	m := make(map[language.Tag]*message.Printer)
	for _, langTag := range supportedTags {
		switch langTag {
		case language.AmericanEnglish, language.English:
			initEnUS(langTag)
		case language.SimplifiedChinese, language.Chinese:
			initZhCN(langTag)
		case language.Japanese:
			initJaJP(langTag)
		case language.TraditionalChinese:
			initZhTW(langTag)
		}
		m[langTag] = message.NewPrinter(langTag)
	}
	p.printers = m
}
