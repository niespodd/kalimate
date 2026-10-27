package ui

import (
	"fmt"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

const (
	meterTopPad  = 20
	meterSidePad = 32
	meterLabelH  = 18
	meterGap     = 10
	meterDrawH   = 96
)

// Meter is the cents gauge. It is live only when the heard key matches the
// target; otherwise the needle is hidden. The needle animates to new positions.
type Meter struct {
	widget.BaseWidget
	show  bool
	cents float64
	zone  zoneKind

	animCents float64
	haveAnim  bool
	anim      *fyne.Animation
	renderer  *meterRenderer
}

func NewMeter() *Meter {
	m := &Meter{}
	m.ExtendBaseWidget(m)
	return m
}

// SetState shows/hides the needle and animates it to cents (relative to target).
func (m *Meter) SetState(show bool, cents float64, z zoneKind) {
	m.show, m.zone = show, z
	target := math.Max(-50, math.Min(50, cents))

	switch {
	case !show:
		// Hidden: stop any sweep and forget the origin so the needle starts AT
		// the reading when it reappears, rather than animating in from 0.
		if m.anim != nil {
			m.anim.Stop()
		}
		m.haveAnim = false
	case !m.haveAnim:
		m.animCents = target
		m.haveAnim = true
	case math.Abs(target-m.animCents) > 0.5:
		from := m.animCents
		if m.anim != nil {
			m.anim.Stop()
		}
		m.anim = fyne.NewAnimation(80*time.Millisecond, func(f float32) {
			m.animCents = from + (target-from)*float64(f)
			if m.renderer != nil {
				m.renderer.placeNeedle(m.Size())
				canvas.Refresh(m)
			}
		})
		m.anim.Start()
	default:
		m.animCents = target
	}
	m.cents = cents
	m.Refresh()
}

func (m *Meter) CreateRenderer() fyne.WidgetRenderer {
	r := &meterRenderer{m: m}
	r.flat = newBoldText("♭ flat", fyne.TextAlignLeading, 13)
	r.center = newBoldText("in tune", fyne.TextAlignCenter, 13)
	r.sharp = newBoldText("♯ sharp", fyne.TextAlignTrailing, 13)

	r.green = canvas.NewRectangle(nil)
	r.green.CornerRadius = 6
	r.green.StrokeWidth = 1
	r.base = canvas.NewRectangle(nil)

	for i := -50; i <= 50; i += 5 {
		r.ticks = append(r.ticks, canvas.NewRectangle(nil))
	}
	for _, v := range []int{-50, -25, 0, 25, 50} {
		t := newBoldText(fmt.Sprintf("%+d", v), fyne.TextAlignCenter, 12)
		if v == 0 {
			t.Text = "0"
		}
		r.tickLabels = append(r.tickLabels, t)
	}

	r.halo = canvas.NewRectangle(nil)
	r.halo.CornerRadius = 5
	r.needle = canvas.NewRectangle(nil)
	r.needle.CornerRadius = 3
	r.overL = newBoldText("‹", fyne.TextAlignCenter, 22)
	r.overR = newBoldText("›", fyne.TextAlignCenter, 22)
	m.renderer = r
	return r
}

type meterRenderer struct {
	m                   *Meter
	flat, center, sharp *canvas.Text
	green               *canvas.Rectangle
	base                *canvas.Rectangle
	ticks               []*canvas.Rectangle
	tickLabels          []*canvas.Text
	halo                *canvas.Rectangle
	needle              *canvas.Rectangle
	overL, overR        *canvas.Text
}

func (r *meterRenderer) xOf(cents float64, w float32) float32 {
	return meterSidePad + w*float32(50+cents)/100
}

func (r *meterRenderer) Layout(size fyne.Size) {
	w := size.Width - 2*meterSidePad

	r.flat.Move(fyne.NewPos(meterSidePad, meterTopPad))
	r.flat.Resize(fyne.NewSize(w, meterLabelH))
	r.center.Move(fyne.NewPos(meterSidePad, meterTopPad))
	r.center.Resize(fyne.NewSize(w, meterLabelH))
	r.sharp.Move(fyne.NewPos(meterSidePad, meterTopPad))
	r.sharp.Resize(fyne.NewSize(w, meterLabelH))

	top := float32(meterTopPad + meterLabelH + meterGap)

	gx0, gx1 := r.xOf(-InTuneCents, w), r.xOf(InTuneCents, w)
	r.green.Move(fyne.NewPos(gx0, top))
	r.green.Resize(fyne.NewSize(gx1-gx0, 72))

	r.base.Move(fyne.NewPos(meterSidePad, top+56))
	r.base.Resize(fyne.NewSize(w, 2))

	for i, t := range r.ticks {
		cents := float64(-50 + i*5)
		x := r.xOf(cents, w)
		major := int(cents)%25 == 0
		var y0, y1 float32 = 51, 61
		if major {
			y0, y1 = 46, 66
		}
		t.Move(fyne.NewPos(x-1, top+y0))
		t.Resize(fyne.NewSize(2, y1-y0))
	}

	majors := []float64{-50, -25, 0, 25, 50}
	for i, lbl := range r.tickLabels {
		x := r.xOf(majors[i], w)
		lbl.Resize(fyne.NewSize(60, 16))
		lbl.Move(fyne.NewPos(x-30, top+74))
	}

	r.placeNeedle(size)
}

// placeNeedle positions the needle, halo, and overflow arrows from animCents.
func (r *meterRenderer) placeNeedle(size fyne.Size) {
	w := size.Width - 2*meterSidePad
	top := float32(meterTopPad + meterLabelH + meterGap)
	c := math.Max(-50, math.Min(50, r.m.animCents))
	nx := r.xOf(c, w)
	r.needle.Move(fyne.NewPos(nx-2.5, top+4))
	r.needle.Resize(fyne.NewSize(5, 64))
	r.halo.Move(fyne.NewPos(nx-5.5, top+1))
	r.halo.Resize(fyne.NewSize(11, 70))

	r.overL.Move(fyne.NewPos(meterSidePad-4, top+20))
	r.overL.Resize(fyne.NewSize(18, 30))
	r.overR.Move(fyne.NewPos(size.Width-meterSidePad-14, top+20))
	r.overR.Resize(fyne.NewSize(18, 30))
}

func (r *meterRenderer) Refresh() {
	pal := currentPalette()
	for _, t := range []*canvas.Text{r.flat, r.center, r.sharp} {
		t.Color = pal.meterLabel
	}
	for _, l := range r.tickLabels {
		l.Color = pal.tickLabel
	}

	fillA, strokeA := 0.12, 0.35
	if r.m.show && r.m.zone == zoneInTune {
		fillA, strokeA = 0.18, 0.50
	}
	r.green.FillColor = withAlpha(pal.needleInTune, fillA)
	r.green.StrokeColor = withAlpha(pal.needleInTune, strokeA)
	r.base.FillColor = pal.baseline
	for i, t := range r.ticks {
		if (-50+i*5)%25 == 0 {
			t.FillColor = pal.majorTick
		} else {
			t.FillColor = pal.minorTick
		}
	}

	nc := pal.needleColor(r.m.zone)
	if r.m.show {
		r.needle.Show()
		r.halo.Show()
		r.needle.FillColor = nc
		r.halo.FillColor = withAlpha(nc, 0.22)
	} else {
		r.needle.Hide()
		r.halo.Hide()
	}

	// Overflow arrows when the reading runs past the scale.
	r.overL.Hide()
	r.overR.Hide()
	if r.m.show {
		if r.m.cents > 50 {
			r.overR.Show()
			r.overR.Color = nc
		} else if r.m.cents < -50 {
			r.overL.Show()
			r.overL.Color = nc
		}
	}

	r.Layout(r.m.Size())
	canvas.Refresh(r.m)
}

func (r *meterRenderer) MinSize() fyne.Size {
	return fyne.NewSize(2*meterSidePad+300, meterTopPad+meterLabelH+meterGap+meterDrawH)
}

func (r *meterRenderer) Objects() []fyne.CanvasObject {
	objs := []fyne.CanvasObject{r.flat, r.center, r.sharp, r.green, r.base}
	for _, t := range r.ticks {
		objs = append(objs, t)
	}
	for _, l := range r.tickLabels {
		objs = append(objs, l)
	}
	return append(objs, r.halo, r.needle, r.overL, r.overR)
}

func (r *meterRenderer) Destroy() {}

func newBoldText(s string, align fyne.TextAlign, size float32) *canvas.Text {
	t := canvas.NewText(s, nil)
	t.TextSize = size
	t.TextStyle = fyne.TextStyle{Bold: true}
	t.Alignment = align
	return t
}
