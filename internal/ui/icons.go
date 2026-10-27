package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// micSVG is a filled microphone glyph. Wrapped in a themed resource so it
// follows the foreground color.
var micSVG = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="#444444">
  <rect x="9" y="2" width="6" height="12" rx="3"/>
  <path d="M6 11a1 1 0 0 1 2 0 4 4 0 0 0 8 0 1 1 0 0 1 2 0 6 6 0 0 1-5 5.92V20h1.5a1 1 0 0 1 0 2h-5a1 1 0 0 1 0-2H11v-3.08A6 6 0 0 1 6 11z"/>
</svg>`)

var micIcon = theme.NewThemedResource(fyne.NewStaticResource("mic.svg", micSVG))
