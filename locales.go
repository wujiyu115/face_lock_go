package main

import (
	"embed"
	"encoding/json"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	localizer *i18n.Localizer
	//go:embed locales/*.json
	localeFS embed.FS
)

func initLocalizer(langs ...string) *i18n.Localizer {
	// Create a new i18n bundle with English as default language.
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	bundle.LoadMessageFileFS(localeFS, "locales/en.json")
	bundle.LoadMessageFileFS(localeFS, "locales/zh-Hans-CN.json")
	// Initialize localizer which will look for phrase keys in passed languages
	// in a strict order (first language is searched first)
	// When no key in any of the languages is found, it fallbacks to default - English language
	localizer := i18n.NewLocalizer(bundle, langs...)
	return localizer
}

func initLocalizers() {
	localizer = initLocalizer(
		language.Chinese.String(),
		language.AmericanEnglish.String(),
	)
}

func getByMessageID(msgId string) string {
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID: msgId,
	})
	checkIfError(err)
	return msg
}
