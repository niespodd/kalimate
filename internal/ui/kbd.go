package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

const (
	kbdHeight = 16
	kbdPadX   = 5
	kbdTextSz = 10
)

// kbdChip is a small keycap-style hint — a rounded, bordered rectangle with a
// centered glyph (e.g. "Space", "→") — used to advertise a keyboard shortcut
// next to the control it triggers.
type kbdChip struct {
	bg   *canvas.Rectangle
	text *canvas.Text
}

func newKbdChip(glyph string) *kbdChip {
	c := &kbdChip{}
	c.bg = canvas.NewRectangle(nil)
	c.bg.CornerRadius = 4
	c.bg.StrokeWidth = 1
	c.text = canvas.NewText(glyph, nil)
	c.text.TextSize = kbdTextSz
	c.text.Alignment = fyne.TextAlignCenter
	c.text.TextStyle = fyne.TextStyle{Bold: true}
	return c
}

// width is the chip's full width at its current glyph.
func (c *kbdChip) width() float32 { return c.text.MinSize().Width + 2*kbdPadX }

// layout positions the chip with its left edge at x, vertically centered on cy.
func (c *kbdChip) layout(x, cy float32) {
	w := c.width()
	c.bg.Move(fyne.NewPos(x, cy-kbdHeight/2))
	c.bg.Resize(fyne.NewSize(w, kbdHeight))
	th := c.text.MinSize().Height
	c.text.Move(fyne.NewPos(x, cy-th/2))
	c.text.Resize(fyne.NewSize(w, th))
}

func (c *kbdChip) applyColors(pal palette) {
	c.bg.FillColor = pal.kbdFill
	c.bg.StrokeColor = pal.kbdBorder
	c.text.Color = pal.kbdText
}

func (c *kbdChip) objects() []fyne.CanvasObject { return []fyne.CanvasObject{c.bg, c.text} }

func (c *kbdChip) show() { c.bg.Show(); c.text.Show() }
func (c *kbdChip) hide() { c.bg.Hide(); c.text.Hide() }
