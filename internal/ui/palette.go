package ui

import (
	"image/color"
	"math"
)

// Tine is one kalimba key: note name, MIDI number (drives tine length), and its
// target frequency. Ordered left→right as they sit on the instrument.
type Tine struct {
	Name   string
	Midi   int
	FreqHz float64
}

func freqOfMidi(midi int) float64 { return 440 * math.Pow(2, float64(midi-69)/12) }

// Layout is the standard 17-key kalimba in C major, physical left→right order.
var Layout = buildLayout()

func buildLayout() []Tine {
	raw := []struct {
		name string
		midi int
	}{
		{"D6", 86}, {"B5", 83}, {"G5", 79}, {"E5", 76}, {"C5", 72}, {"A4", 69}, {"F4", 65}, {"D4", 62},
		{"C4", 60},
		{"E4", 64}, {"G4", 67}, {"B4", 71}, {"D5", 74}, {"F5", 77}, {"A5", 81}, {"C6", 84}, {"E6", 88},
	}
	out := make([]Tine, len(raw))
	for i, r := range raw {
		out[i] = Tine{Name: r.name, Midi: r.midi, FreqHz: freqOfMidi(r.midi)}
	}
	return out
}

var tineByName = func() map[string]Tine {
	m := make(map[string]Tine, len(Layout))
	for _, t := range Layout {
		m[t.Name] = t
	}
	return m
}()

// TineByName looks up a tine by its note name.
func TineByName(name string) (Tine, bool) {
	t, ok := tineByName[name]
	return t, ok
}

// Tuning zones.
const (
	InTuneCents = 5.0  // ±5¢ = in tune
	CloseCents  = 20.0 // ≤20¢ = close
)

type zoneKind int

const (
	zoneIdle zoneKind = iota
	zoneInTune
	zoneClose
	zoneFar
)

func zoneOf(cents float64, hasSignal bool) zoneKind {
	if !hasSignal {
		return zoneIdle
	}
	switch a := math.Abs(cents); {
	case a <= InTuneCents:
		return zoneInTune
	case a <= CloseCents:
		return zoneClose
	default:
		return zoneFar
	}
}

func rgb(r, g, b uint8) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: 255} }

// withAlpha returns c at the given alpha fraction (0..1).
func withAlpha(c color.Color, frac float64) color.NRGBA {
	r, g, b, _ := c.RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(frac * 255)}
}

// palette holds every semantic color the view uses, so a widget restyles by
// re-reading it on Refresh.
type palette struct {
	toolbarBg    color.Color
	separator    color.Color
	title        color.Color
	secondary    color.Color
	levelLit     color.Color
	levelUnlit   color.Color
	micFill      color.Color
	micFillHover color.Color // control fill under the mouse (mic picker, button)
	micBorder    color.Color

	notePrimary color.Color
	idleText    color.Color

	meterLabel   color.Color
	greenFill    color.Color
	greenStroke  color.Color
	baseline     color.Color
	majorTick    color.Color
	minorTick    color.Color
	tickLabel    color.Color
	needleIdle   color.Color
	needleInTune color.Color
	needleClose  color.Color
	needleFar    color.Color

	dirIdle   color.Color
	dirInTune color.Color
	dirClose  color.Color
	dirFar    color.Color

	kalCaption     color.Color
	kalBody        color.Color
	kalBridge      color.Color
	kalNeutral     color.Color
	kalDone        color.Color
	kalPending     color.Color
	kalActive      color.Color
	kalActiveTuned color.Color
	targetWaiting  color.Color // target tine with no signal / wrong key
	ring           color.Color // target selection ring
	heardOff       color.Color // "what I hear" dot, off-target
	heardTuned     color.Color // "what I hear" dot, on-target & in tune
	micBad         color.Color // "Microphone unavailable"

	lblPending color.Color
	lblDone    color.Color
	lblTarget  color.Color

	kalHover  color.Color // faint chip behind the label of a hovered key
	kbdFill   color.Color // keycap hint background
	kbdBorder color.Color // keycap hint border
	kbdText   color.Color // keycap hint glyph
}

var lightPalette = palette{
	toolbarBg:    rgb(0xF5, 0xF5, 0xF7),
	separator:    rgb(0xE5, 0xE5, 0xEA),
	title:        rgb(0x6E, 0x6E, 0x73),
	secondary:    rgb(0x6E, 0x6E, 0x73),
	levelLit:     rgb(0x3A, 0x3A, 0x3C),
	levelUnlit:   rgb(0xD1, 0xD1, 0xD6),
	micFill:      rgb(0xFF, 0xFF, 0xFF),
	micFillHover: rgb(0xEC, 0xEC, 0xF0),
	micBorder:    rgb(0xD1, 0xD1, 0xD6),

	notePrimary: rgb(0x1D, 0x1D, 0x1F),
	idleText:    rgb(0x8E, 0x8E, 0x93),

	meterLabel:   rgb(0x6E, 0x6E, 0x73),
	greenFill:    withAlpha(rgb(0x15, 0x80, 0x3D), 0.12),
	greenStroke:  withAlpha(rgb(0x15, 0x80, 0x3D), 0.35),
	baseline:     rgb(0xD1, 0xD1, 0xD6),
	majorTick:    rgb(0x8E, 0x8E, 0x93),
	minorTick:    rgb(0xC7, 0xC7, 0xCC),
	tickLabel:    rgb(0x6E, 0x6E, 0x73),
	needleIdle:   rgb(0xC7, 0xC7, 0xCC),
	needleInTune: rgb(0x15, 0x80, 0x3D),
	needleClose:  rgb(0xD9, 0x77, 0x06),
	needleFar:    rgb(0xB9, 0x1C, 0x1C),

	dirIdle:   rgb(0x6E, 0x6E, 0x73),
	dirInTune: rgb(0x15, 0x80, 0x3D),
	dirClose:  rgb(0xB4, 0x53, 0x09),
	dirFar:    rgb(0xB9, 0x1C, 0x1C),

	kalCaption:     rgb(0x6E, 0x6E, 0x73),
	kalBody:        rgb(0xEB, 0xD5, 0xB3),
	kalBridge:      rgb(0x6B, 0x4B, 0x2A),
	kalNeutral:     rgb(0x4B, 0x4F, 0x58),
	kalDone:        rgb(0x15, 0x80, 0x3D),
	kalPending:     rgb(0x7D, 0x80, 0x89),
	kalActive:      rgb(0xD9, 0x77, 0x06),
	kalActiveTuned: rgb(0x15, 0x80, 0x3D),
	targetWaiting:  rgb(0x3A, 0x3A, 0x3C),
	ring:           rgb(0x1D, 0x1D, 0x1F),
	heardOff:       rgb(0xB4, 0x53, 0x09),
	heardTuned:     rgb(0x15, 0x80, 0x3D),
	micBad:         rgb(0xB9, 0x1C, 0x1C),

	lblPending: rgb(0x5C, 0x4A, 0x33),
	lblDone:    rgb(0x14, 0x53, 0x2D),
	lblTarget:  rgb(0x1D, 0x1D, 0x1F),

	kalHover:  withAlpha(color.White, 0.40),
	kbdFill:   rgb(0xFF, 0xFF, 0xFF),
	kbdBorder: rgb(0xC7, 0xC7, 0xCC),
	kbdText:   rgb(0x6E, 0x6E, 0x73),
}

// currentPalette returns the active palette. The app is pinned to light, so
// this is constant; kept as a function so widgets read it uniformly on Refresh.
func currentPalette() palette { return lightPalette }

func (p palette) needleColor(z zoneKind) color.Color {
	switch z {
	case zoneInTune:
		return p.needleInTune
	case zoneClose:
		return p.needleClose
	case zoneFar:
		return p.needleFar
	default:
		return p.needleIdle
	}
}

func (p palette) dirColor(z zoneKind) color.Color {
	switch z {
	case zoneInTune:
		return p.dirInTune
	case zoneClose:
		return p.dirClose
	case zoneFar:
		return p.dirFar
	default:
		return p.dirIdle
	}
}

// centsColor is the hero cents-line color: zone-driven, muted when idle.
func (p palette) centsColor(z zoneKind) color.Color {
	switch z {
	case zoneInTune:
		return p.needleInTune
	case zoneClose:
		return p.needleClose
	case zoneFar:
		return p.needleFar
	default:
		return p.idleText
	}
}

// targetColor is the fill of the selected target tine, by tuning zone. zoneIdle
// (no signal or wrong key) is the neutral "waiting" color.
func (p palette) targetColor(z zoneKind) color.Color {
	switch z {
	case zoneInTune:
		return p.kalActiveTuned
	case zoneClose:
		return p.kalActive
	case zoneFar:
		return p.needleFar
	default:
		return p.targetWaiting
	}
}
