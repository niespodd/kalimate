package ui

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// adviceHold keeps "Slide the tine up/down" on screen at least this long after
// the pluck stops ringing, so the direction doesn't vanish mid-adjustment.
const adviceHold = 3 * time.Second

const (
	dirTopPad   = 16
	dirHeight   = 60
	dirIconSize = 28
	dirIconGap  = 12
	dirStroke   = 2.2
	dirBtnH     = 36
	dirBtnPad   = 12
	dirBtnGap   = 8
	dirBtnArrow = 16
	dirKbdGap   = 8
)

// Directive tells the player what to do. In tune, it also offers a "Next"
// button (tappable) that advances to the next key.
type Directive struct {
	widget.BaseWidget
	// OnNext fires when the Next / Start over button is tapped.
	OnNext func()

	ph         phase
	cents      float64
	zone       zoneKind
	target     string
	heard      string
	nextKey    string
	allDone    bool
	micError   bool
	targetDone bool

	btnPos   fyne.Position
	btnSize  fyne.Size
	hoverBtn bool

	now         func() time.Time
	adviceUntil time.Time
	adviceCents float64
	adviceZone  zoneKind
}

func NewDirective() *Directive {
	d := &Directive{now: time.Now}
	d.ExtendBaseWidget(d)
	return d
}

func (d *Directive) SetState(ph phase, cents float64, z zoneKind, target, heard, nextKey string, allDone, micError, targetDone bool) {
	now := d.now()
	if target != d.target {
		d.adviceUntil = time.Time{} // new key: forget the previous slide hint
	}

	// Latch push/pull advice so it survives the ~600 ms gap after the tine stops
	// ringing: sliding a tine takes longer than that, and losing the direction
	// mid-adjustment is disorienting. Hold it for adviceHold — refreshed while the
	// pluck keeps sounding — but drop it the instant a newer reading (in tune, or
	// a different key) makes it stale.
	switch {
	case !micError && ph == phaseMatch && z != zoneInTune:
		d.adviceCents, d.adviceZone = cents, z
		d.adviceUntil = now.Add(adviceHold)
	case ph == phaseWrongKey || (ph == phaseMatch && z == zoneInTune):
		d.adviceUntil = time.Time{}
	case !micError && ph == phaseNoSignal && !targetDone && now.Before(d.adviceUntil):
		// Signal dropped, advice still fresh: keep showing it instead of falling
		// back to the "Pluck the key" prompt.
		ph, cents, z = phaseMatch, d.adviceCents, d.adviceZone
	}

	d.ph, d.cents, d.zone = ph, cents, z
	d.target, d.heard, d.nextKey = target, heard, nextKey
	d.allDone, d.micError, d.targetDone = allDone, micError, targetDone
	d.Refresh()
}

func (d *Directive) showsButton() bool {
	if d.micError {
		return false
	}
	// In tune right now, or already confirmed Done — so the button stays
	// reachable after the tine stops ringing (bug 13).
	return (d.ph == phaseMatch && d.zone == zoneInTune) || d.targetDone
}

func (d *Directive) overButton(pos fyne.Position) bool {
	return d.showsButton() &&
		pos.X >= d.btnPos.X && pos.X <= d.btnPos.X+d.btnSize.Width &&
		pos.Y >= d.btnPos.Y && pos.Y <= d.btnPos.Y+d.btnSize.Height
}

func (d *Directive) Tapped(ev *fyne.PointEvent) {
	if d.OnNext == nil {
		return
	}
	if d.overButton(ev.Position) {
		d.OnNext()
	}
}

// MouseIn, MouseMoved, MouseOut and Cursor give the Next button a pointer
// cursor and a hover fill, so it reads as clickable.
func (d *Directive) MouseIn(ev *desktop.MouseEvent)    { d.setHoverBtn(d.overButton(ev.Position)) }
func (d *Directive) MouseMoved(ev *desktop.MouseEvent) { d.setHoverBtn(d.overButton(ev.Position)) }
func (d *Directive) MouseOut()                         { d.setHoverBtn(false) }

func (d *Directive) setHoverBtn(over bool) {
	if over != d.hoverBtn {
		d.hoverBtn = over
		d.Refresh()
	}
}

func (d *Directive) Cursor() desktop.Cursor {
	if d.hoverBtn {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (d *Directive) CreateRenderer() fyne.WidgetRenderer {
	mk := func() *canvas.Line { l := canvas.NewLine(nil); l.StrokeWidth = dirStroke; return l }
	r := &directiveRenderer{d: d}
	r.text = canvas.NewText("", nil)
	r.text.Alignment = fyne.TextAlignLeading
	r.bar, r.shaft, r.head1, r.head2 = mk(), mk(), mk(), mk()
	r.check1, r.check2, r.bang = mk(), mk(), mk()
	r.circle = canvas.NewCircle(color.Transparent)
	r.circle.StrokeWidth = dirStroke
	r.bangDot = canvas.NewCircle(nil)

	r.btnBg = canvas.NewRectangle(nil)
	r.btnBg.CornerRadius = 8
	r.btnBg.StrokeWidth = 1
	r.btnText = canvas.NewText("", nil)
	r.btnText.TextSize = 14
	r.btnText.TextStyle = fyne.TextStyle{Bold: true}
	r.btnArrow, r.btnAh1, r.btnAh2 = mk(), mk(), mk()
	r.kbd = newKbdChip("Space")
	return r
}

type directiveRenderer struct {
	d    *Directive
	text *canvas.Text

	bar, shaft, head1, head2 *canvas.Line
	check1, check2, bang     *canvas.Line
	circle, bangDot          *canvas.Circle

	btnBg                    *canvas.Rectangle
	btnText                  *canvas.Text
	btnArrow, btnAh1, btnAh2 *canvas.Line
	kbd                      *kbdChip
}

type dirIcon int

const (
	icNone  dirIcon = iota
	icPull          // bar + down arrow
	icPush          // bar + up arrow
	icBang          // circle + "!"
	icCheck         // circle + check
)

// content returns the icon, text, size, and bold flag for the current state.
func (r *directiveRenderer) content() (dirIcon, string, float32, bool) {
	d := r.d
	if d.micError {
		return icNone, tr("micPermission"), 20, false
	}
	switch d.ph {
	case phaseNoTarget:
		return icNone, tr("tapKeyBelow"), 20, false
	case phaseNoSignal:
		if d.targetDone { // tuned, then the ring faded — keep the confirmation up
			return icCheck, tr("inTune"), 26, true
		}
		return icNone, tr("pluckKey", map[string]any{"Key": d.target}), 24, true
	case phaseWrongKey:
		return icBang, tr("wrongKey", map[string]any{"Heard": d.heard, "Target": d.target}), 24, true
	default: // phaseMatch
		switch d.zone {
		case zoneInTune:
			return icCheck, tr("inTune"), 26, true
		default:
			// The on-screen tines hang down from the bridge; a longer free length
			// sounds lower. Sharp → lengthen (slide down); flat → shorten (slide up).
			if d.cents > 0 {
				return icPull, tr("pullTineOut"), 24, true
			}
			return icPush, tr("pushTineIn"), 24, true
		}
	}
}

func (r *directiveRenderer) textColor() color.Color {
	pal := currentPalette()
	d := r.d
	if d.micError {
		return pal.micBad
	}
	switch d.ph {
	case phaseNoSignal:
		if d.targetDone { // matches the "In tune" confirmation held after the ring
			return pal.dirColor(zoneInTune)
		}
		return pal.secondary
	case phaseNoTarget:
		return pal.secondary
	case phaseWrongKey:
		return pal.dirClose
	default:
		return pal.dirColor(d.zone)
	}
}

func (r *directiveRenderer) Layout(size fyne.Size) {
	ic, txt, sz, bold := r.content()
	r.text.Text = txt
	r.text.TextSize = sz
	if bold {
		r.text.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		r.text.TextStyle = fyne.TextStyle{}
	}
	textW := r.text.MinSize().Width
	textH := r.text.MinSize().Height
	midY := float32(dirTopPad + dirHeight/2)

	blockW := textW
	if ic != icNone {
		blockW += dirIconSize + dirIconGap
	}
	btnW := float32(0)
	kbdW := float32(0)
	if r.d.showsButton() {
		r.btnText.Text = r.buttonLabel()
		btnW = dirBtnPad + r.btnText.MinSize().Width + dirBtnGap + dirBtnArrow + dirBtnPad
		kbdW = dirKbdGap + r.kbd.width()
		blockW += dirIconGap + btnW + kbdW
	}

	startX := (size.Width - blockW) / 2
	x := startX
	if ic != icNone {
		r.layoutIcon(ic, x, midY)
		x += dirIconSize + dirIconGap
	}
	r.text.Move(fyne.NewPos(x, midY-textH/2))
	r.text.Resize(fyne.NewSize(textW+4, textH))
	x += textW + dirIconGap

	if r.d.showsButton() {
		r.layoutButton(x, midY, btnW)
		r.kbd.layout(x+btnW+dirKbdGap, midY)
	}
}

func (r *directiveRenderer) buttonLabel() string {
	if r.d.allDone {
		return tr("startOver")
	}
	return tr("nextKey", map[string]any{"Key": r.d.nextKey})
}

func (r *directiveRenderer) layoutButton(x, cy, w float32) {
	top := cy - dirBtnH/2
	r.btnBg.Move(fyne.NewPos(x, top))
	r.btnBg.Resize(fyne.NewSize(w, dirBtnH))
	r.d.btnPos = fyne.NewPos(x, top)
	r.d.btnSize = fyne.NewSize(w, dirBtnH)

	th := r.btnText.MinSize().Height
	r.btnText.Move(fyne.NewPos(x+dirBtnPad, cy-th/2))
	r.btnText.Resize(fyne.NewSize(r.btnText.MinSize().Width+2, th))

	if r.d.allDone {
		r.btnArrow.Hide()
		r.btnAh1.Hide()
		r.btnAh2.Hide()
		return
	}
	ax := x + w - dirBtnPad - dirBtnArrow
	r.btnArrow.Show()
	r.btnAh1.Show()
	r.btnAh2.Show()
	r.btnArrow.Position1 = fyne.NewPos(ax, cy)
	r.btnArrow.Position2 = fyne.NewPos(ax+dirBtnArrow, cy)
	tip := fyne.NewPos(ax+dirBtnArrow, cy)
	r.btnAh1.Position1, r.btnAh1.Position2 = tip, fyne.NewPos(ax+dirBtnArrow-5, cy-4)
	r.btnAh2.Position1, r.btnAh2.Position2 = tip, fyne.NewPos(ax+dirBtnArrow-5, cy+4)
}

func (r *directiveRenderer) layoutIcon(ic dirIcon, x, cy float32) {
	for _, o := range []*canvas.Line{r.bar, r.shaft, r.head1, r.head2, r.check1, r.check2, r.bang} {
		o.Hide()
	}
	r.circle.Hide()
	r.bangDot.Hide()

	cx := x + dirIconSize/2
	top := cy - dirIconSize/2
	bot := cy + dirIconSize/2

	switch ic {
	case icPull, icPush:
		r.bar.Show()
		r.bar.Position1 = fyne.NewPos(x+4, top+3)
		r.bar.Position2 = fyne.NewPos(x+dirIconSize-4, top+3)
		r.shaft.Show()
		r.shaft.Position1 = fyne.NewPos(cx, top+8)
		r.shaft.Position2 = fyne.NewPos(cx, bot-2)
		r.head1.Show()
		r.head2.Show()
		if ic == icPull {
			tip := fyne.NewPos(cx, bot-2)
			r.head1.Position1, r.head1.Position2 = tip, fyne.NewPos(cx-5, bot-8)
			r.head2.Position1, r.head2.Position2 = tip, fyne.NewPos(cx+5, bot-8)
		} else {
			tip := fyne.NewPos(cx, top+8)
			r.head1.Position1, r.head1.Position2 = tip, fyne.NewPos(cx-5, top+14)
			r.head2.Position1, r.head2.Position2 = tip, fyne.NewPos(cx+5, top+14)
		}
	case icCheck:
		r.circle.Show()
		r.circle.Position1 = fyne.NewPos(x+2, top+2)
		r.circle.Position2 = fyne.NewPos(x+dirIconSize-2, bot-2)
		r.check1.Show()
		r.check2.Show()
		r.check1.Position1 = fyne.NewPos(cx-6, cy+1)
		r.check1.Position2 = fyne.NewPos(cx-1, cy+6)
		r.check2.Position1 = fyne.NewPos(cx-1, cy+6)
		r.check2.Position2 = fyne.NewPos(cx+7, cy-6)
	case icBang:
		r.circle.Show()
		r.circle.Position1 = fyne.NewPos(x+2, top+2)
		r.circle.Position2 = fyne.NewPos(x+dirIconSize-2, bot-2)
		r.bang.Show()
		r.bang.Position1 = fyne.NewPos(cx, cy-6)
		r.bang.Position2 = fyne.NewPos(cx, cy+2)
		r.bangDot.Show()
		r.bangDot.Position1 = fyne.NewPos(cx-1.3, cy+5)
		r.bangDot.Position2 = fyne.NewPos(cx+1.3, cy+7.6)
	}
}

func (r *directiveRenderer) Refresh() {
	pal := currentPalette()
	c := r.textColor()
	r.text.Color = c
	for _, l := range []*canvas.Line{r.bar, r.shaft, r.head1, r.head2, r.check1, r.check2, r.bang} {
		l.StrokeColor = c
	}
	r.circle.StrokeColor = c
	r.bangDot.FillColor = c

	if r.d.showsButton() {
		r.btnBg.Show()
		r.btnText.Show()
		if r.d.hoverBtn {
			r.btnBg.FillColor = pal.micFillHover
		} else {
			r.btnBg.FillColor = pal.micFill
		}
		r.btnBg.StrokeColor = pal.micBorder
		r.btnText.Color = pal.notePrimary
		for _, l := range []*canvas.Line{r.btnArrow, r.btnAh1, r.btnAh2} {
			l.StrokeColor = pal.notePrimary
		}
		r.kbd.show()
		r.kbd.applyColors(pal)
	} else {
		r.btnBg.Hide()
		r.btnText.Hide()
		r.btnArrow.Hide()
		r.btnAh1.Hide()
		r.btnAh2.Hide()
		r.kbd.hide()
	}

	r.Layout(r.d.Size())
	canvas.Refresh(r.d)
}

func (r *directiveRenderer) MinSize() fyne.Size {
	return fyne.NewSize(300, dirTopPad+dirHeight)
}

func (r *directiveRenderer) Objects() []fyne.CanvasObject {
	objs := []fyne.CanvasObject{
		r.text, r.bar, r.shaft, r.head1, r.head2, r.circle, r.check1, r.check2,
		r.bang, r.bangDot, r.btnBg, r.btnText, r.btnArrow, r.btnAh1, r.btnAh2,
	}
	return append(objs, r.kbd.objects()...)
}

func (r *directiveRenderer) Destroy() {}
