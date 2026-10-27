package ui

import (
	"fmt"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

const (
	heroTopMargin = 36
	heroBlock     = 150
	heroGap       = 6
)

// NoteDisplay is the hero readout. With no target it prompts "Choose a key";
// with a target it shows a "TUNING" overline, the target name, and a status
// line (cents / "Hearing X" / "Pluck it"). The block height is fixed so the
// meter below never moves.
type NoteDisplay struct {
	widget.BaseWidget
	phase  phase
	target string
	heard  string
	cents  float64
	zone   zoneKind
}

func NewNoteDisplay() *NoteDisplay {
	d := &NoteDisplay{}
	d.ExtendBaseWidget(d)
	return d
}

func (d *NoteDisplay) SetState(ph phase, target, heard string, cents float64, z zoneKind) {
	d.phase, d.target, d.heard, d.cents, d.zone = ph, target, heard, cents, z
	d.Refresh()
}

func (d *NoteDisplay) CreateRenderer() fyne.WidgetRenderer {
	r := &noteDisplayRenderer{d: d}
	r.overline = canvas.NewText(tr("tuningOverline"), nil)
	r.overline.TextSize = 12
	r.overline.TextStyle = fyne.TextStyle{Bold: true}
	r.overline.Alignment = fyne.TextAlignCenter
	r.name = canvas.NewText("", nil)
	r.name.TextSize = 96
	r.name.TextStyle = fyne.TextStyle{Bold: true}
	r.name.Alignment = fyne.TextAlignCenter
	r.status = canvas.NewText("", nil)
	r.status.TextSize = 24
	r.status.TextStyle = fyne.TextStyle{Bold: true}
	r.status.Alignment = fyne.TextAlignCenter
	r.idle = canvas.NewText(tr("chooseKey"), nil)
	r.idle.TextSize = 52
	r.idle.TextStyle = fyne.TextStyle{Bold: true}
	r.idle.Alignment = fyne.TextAlignCenter
	return r
}

type noteDisplayRenderer struct {
	d        *NoteDisplay
	overline *canvas.Text
	name     *canvas.Text
	status   *canvas.Text
	idle     *canvas.Text
}

func (r *noteDisplayRenderer) Layout(size fyne.Size) {
	w := size.Width
	if r.d.phase == phaseNoTarget {
		h := r.idle.MinSize().Height
		r.idle.Resize(fyne.NewSize(w, h))
		r.idle.Move(fyne.NewPos(0, heroTopMargin+(heroBlock-h)/2))
		return
	}
	oH := r.overline.MinSize().Height
	nH := r.name.MinSize().Height
	sH := r.status.MinSize().Height
	total := oH + heroGap + nH + heroGap + sH
	top := heroTopMargin + (heroBlock-total)/2

	r.overline.Resize(fyne.NewSize(w, oH))
	r.overline.Move(fyne.NewPos(0, top))
	r.name.Resize(fyne.NewSize(w, nH))
	r.name.Move(fyne.NewPos(0, top+oH+heroGap))
	r.status.Resize(fyne.NewSize(w, sH))
	r.status.Move(fyne.NewPos(0, top+oH+heroGap+nH+heroGap))
}

func (r *noteDisplayRenderer) Refresh() {
	pal := currentPalette()
	if r.d.phase == phaseNoTarget {
		r.idle.Show()
		r.idle.Color = pal.idleText
		r.overline.Hide()
		r.name.Hide()
		r.status.Hide()
		r.Layout(r.d.Size())
		canvas.Refresh(r.d)
		return
	}
	r.idle.Hide()
	r.overline.Show()
	r.name.Show()
	r.status.Show()

	r.overline.Color = pal.secondary
	r.name.Text = r.d.target
	r.name.Color = pal.notePrimary

	switch r.d.phase {
	case phaseMatch:
		r.status.Text = formatCents(r.d.cents, r.d.zone)
		r.status.Color = pal.centsColor(r.d.zone)
	case phaseWrongKey:
		r.status.Text = tr("hearing", map[string]any{"Heard": r.d.heard})
		r.status.Color = pal.dirClose
	default: // phaseNoSignal
		r.status.Text = tr("pluckIt")
		r.status.Color = pal.secondary
	}

	r.Layout(r.d.Size())
	canvas.Refresh(r.d)
}

func (r *noteDisplayRenderer) MinSize() fyne.Size {
	return fyne.NewSize(360, heroTopMargin+heroBlock)
}

func (r *noteDisplayRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.overline, r.name, r.status, r.idle}
}

func (r *noteDisplayRenderer) Destroy() {}

// formatCents renders "+18¢ sharp" / "−12¢ flat" / "+1¢" (no word in tune).
func formatCents(c float64, z zoneKind) string {
	c = math.Max(-50, math.Min(50, c))
	v := int(math.Round(c))
	sign := "+"
	if v < 0 {
		sign = "−" // U+2212
		v = -v
	}
	s := fmt.Sprintf("%s%d¢", sign, v)
	if z != zoneInTune {
		if c > 0 {
			s += " " + tr("centsSharp")
		} else if c < 0 {
			s += " " + tr("centsFlat")
		}
	}
	return s
}
