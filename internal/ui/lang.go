package ui

import (
	"embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
)

// translations holds the bundled message catalogs. Add further locales by
// dropping a "<lang>.json" file (e.g. "pl.json") into the directory below; the
// file structure follows github.com/nicksnyder/go-i18n, keyed by message ID.
//
//go:embed translations
var translations embed.FS

// loadTranslations registers the bundled catalogs with Fyne's localizer. The
// active locale is resolved from the host system. Call once before any widget
// renderer is created (New does this), since captions are localized at build
// time.
func loadTranslations() {
	if err := lang.AddTranslationsFS(translations, "translations"); err != nil {
		fyne.LogError("could not load translations", err)
	}
}

// tr localizes a message by ID, interpolating the optional template data. It is
// a thin wrapper over lang.X that keeps call sites terse; the English text in
// translations/en.json is the fallback when a locale lacks the key.
func tr(id string, data ...any) string {
	return lang.X(id, id, data...)
}
