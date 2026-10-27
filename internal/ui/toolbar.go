package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

const (
	toolbarH    = 48
	micCtrlH    = 36
	micPad      = 10
	micGap      = 8
	micIconSz   = 16
	micChevronW = 14
	micNameMax  = 220
)

var levelBarHeights = []float32{6, 9, 12, 15, 18}

// Toolbar is the top strip: title, input-level bars, and a labeled mic-picker
// control that opens the device menu.
type Toolbar struct {
	widget.BaseWidget
	onSelect  func(int)
	onRefresh func()
	canvas    fyne.Canvas

	level    float64
	devices  []string
	current  int
	micName  string
	micError bool

	ctrlPos   fyne.Position
	ctrlSize  fyne.Size
	hoverCtrl bool
}

func NewToolbar(c fyne.Canvas) *Toolbar {
	t := &Toolbar{canvas: c}
	t.ExtendBaseWidget(t)
	return t
}

func (t *Toolbar) SetLevel(level float64) { t.level = level; t.Refresh() }

func (t *Toolbar) SetDevices(names []string, current int) {
	t.devices, t.current = names, current
	t.Refresh()
}

func (t *Toolbar) SetMic(name string, bad bool) {
	t.micName, t.micError = name, bad
	t.Refresh()
}

func (t *Toolbar) overCtrl(pos fyne.Position) bool {
	return pos.X >= t.ctrlPos.X && pos.X <= t.ctrlPos.X+t.ctrlSize.Width &&
		pos.Y >= t.ctrlPos.Y && pos.Y <= t.ctrlPos.Y+t.ctrlSize.Height
}

// MouseIn, MouseMoved, MouseOut and Cursor give the mic picker a pointer cursor
// and a hover fill, so it reads as a button rather than static text.
func (t *Toolbar) MouseIn(ev *desktop.MouseEvent)    { t.setHoverCtrl(t.overCtrl(ev.Position)) }
func (t *Toolbar) MouseMoved(ev *desktop.MouseEvent) { t.setHoverCtrl(t.overCtrl(ev.Position)) }
func (t *Toolbar) MouseOut()                         { t.setHoverCtrl(false) }

func (t *Toolbar) setHoverCtrl(over bool) {
	if over != t.hoverCtrl {
		t.hoverCtrl = over
		t.Refresh()
	}
}

func (t *Toolbar) Cursor() desktop.Cursor {
	if t.hoverCtrl {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (t *Toolbar) Tapped(ev *fyne.PointEvent) {
	if !t.overCtrl(ev.Position) {
		return
	}
	if t.canvas == nil {
		return
	}
	items := make([]*fyne.MenuItem, 0, len(t.devices)+2)
	for i, name := range t.devices {
		idx := i
		it := fyne.NewMenuItem(name, func() {
			if t.onSelect != nil {
				t.onSelect(idx)
			}
		})
		it.Checked = i == t.current
		items = append(items, it)
	}
	items = append(items, fyne.NewMenuItemSeparator())
	items = append(items, fyne.NewMenuItem(tr("refreshDevices"), func() {
		if t.onRefresh != nil {
			t.onRefresh()
		}
	}))
	pop := widget.NewPopUpMenu(fyne.NewMenu("", items...), t.canvas)
	pop.ShowAtPosition(fyne.NewPos(ev.AbsolutePosition.X-ev.Position.X+t.ctrlPos.X,
		ev.AbsolutePosition.Y-ev.Position.Y+t.ctrlPos.Y+t.ctrlSize.Height+2))
}

func (t *Toolbar) CreateRenderer() fyne.WidgetRenderer {
	r := &toolbarRenderer{t: t}
	r.fill = canvas.NewRectangle(nil)
	r.sep = canvas.NewRectangle(nil)
	r.title = newBoldText("Kalimate", fyne.TextAlignLeading, 13)
	for range levelBarHeights {
		r.bars = append(r.bars, canvas.NewRectangle(nil))
	}
	r.ctrlBg = canvas.NewRectangle(nil)
	r.ctrlBg.CornerRadius = 8
	r.ctrlBg.StrokeWidth = 1
	r.icon = canvas.NewImageFromResource(micIcon)
	r.icon.FillMode = canvas.ImageFillContain
	r.name = canvas.NewText("", nil)
	r.name.TextSize = 13
	r.chev1 = canvas.NewLine(nil)
	r.chev1.StrokeWidth = 2
	r.chev2 = canvas.NewLine(nil)
	r.chev2.StrokeWidth = 2
	return r
}

type toolbarRenderer struct {
	t            *Toolbar
	fill         *canvas.Rectangle
	sep          *canvas.Rectangle
	title        *canvas.Text
	bars         []*canvas.Rectangle
	ctrlBg       *canvas.Rectangle
	icon         *canvas.Image
	name         *canvas.Text
	chev1, chev2 *canvas.Line
}

func (r *toolbarRenderer) Layout(size fyne.Size) {
	r.fill.Move(fyne.NewPos(0, 0))
	r.fill.Resize(size)
	r.sep.Move(fyne.NewPos(0, size.Height-1))
	r.sep.Resize(fyne.NewSize(size.Width, 1))

	th := r.title.MinSize().Height
	r.title.Move(fyne.NewPos(16, (size.Height-th)/2))
	r.title.Resize(fyne.NewSize(200, th))

	// Mic control, anchored 12 from the right edge.
	label := r.t.micName
	if r.t.micError {
		label = tr("micUnavailable")
	}
	fitText(r.name, label, micNameMax)
	nameW := r.name.MinSize().Width
	ctrlW := micPad + micIconSz + micGap + nameW + micGap + micChevronW + micPad
	ctrlX := size.Width - 12 - ctrlW
	ctrlY := (size.Height - micCtrlH) / 2
	r.ctrlBg.Move(fyne.NewPos(ctrlX, ctrlY))
	r.ctrlBg.Resize(fyne.NewSize(ctrlW, micCtrlH))
	r.t.ctrlPos = fyne.NewPos(ctrlX, ctrlY)
	r.t.ctrlSize = fyne.NewSize(ctrlW, micCtrlH)

	iconX := ctrlX + micPad
	r.icon.Resize(fyne.NewSize(micIconSz, micIconSz))
	r.icon.Move(fyne.NewPos(iconX, (size.Height-micIconSz)/2))
	nameX := iconX + micIconSz + micGap
	nh := r.name.MinSize().Height
	r.name.Move(fyne.NewPos(nameX, (size.Height-nh)/2))
	r.name.Resize(fyne.NewSize(nameW+2, nh))

	// Chevron-down.
	cx := nameX + nameW + micGap + micChevronW/2
	cy := size.Height / 2
	r.chev1.Position1 = fyne.NewPos(cx-5, cy-2)
	r.chev1.Position2 = fyne.NewPos(cx, cy+3)
	r.chev2.Position1 = fyne.NewPos(cx, cy+3)
	r.chev2.Position2 = fyne.NewPos(cx+5, cy-2)

	// Level bars, to the left of the control.
	barsW := float32(len(r.bars)*3 + (len(r.bars)-1)*2)
	barsX := ctrlX - 10 - barsW
	baseline := (size.Height + 18) / 2
	for i, b := range r.bars {
		h := levelBarHeights[i]
		b.CornerRadius = 2
		b.Resize(fyne.NewSize(3, h))
		b.Move(fyne.NewPos(barsX+float32(i)*5, baseline-h))
	}
}

func (r *toolbarRenderer) Refresh() {
	pal := currentPalette()
	r.fill.FillColor = pal.toolbarBg
	r.sep.FillColor = pal.separator
	r.title.Color = pal.title
	if r.t.hoverCtrl {
		r.ctrlBg.FillColor = pal.micFillHover
	} else {
		r.ctrlBg.FillColor = pal.micFill
	}
	r.ctrlBg.StrokeColor = pal.micBorder
	if r.t.micError {
		r.name.Color = pal.micBad
	} else {
		r.name.Color = pal.notePrimary
	}
	r.chev1.StrokeColor = pal.secondary
	r.chev2.StrokeColor = pal.secondary

	lit := int(r.t.level*5 + 0.5)
	for i, b := range r.bars {
		if i < lit {
			b.FillColor = pal.levelLit
		} else {
			b.FillColor = pal.levelUnlit
		}
	}
	r.Layout(r.t.Size())
	canvas.Refresh(r.t)
}

func (r *toolbarRenderer) MinSize() fyne.Size { return fyne.NewSize(420, toolbarH) }

func (r *toolbarRenderer) Objects() []fyne.CanvasObject {
	objs := []fyne.CanvasObject{r.fill, r.sep, r.title}
	for _, b := range r.bars {
		objs = append(objs, b)
	}
	return append(objs, r.ctrlBg, r.icon, r.name, r.chev1, r.chev2)
}

func (r *toolbarRenderer) Destroy() {}

// fitText sets t.Text to s, truncated with an ellipsis to fit maxW.
func fitText(t *canvas.Text, s string, maxW float32) {
	t.Text = s
	if t.MinSize().Width <= maxW {
		return
	}
	runes := []rune(s)
	for len(runes) > 1 {
		runes = runes[:len(runes)-1]
		t.Text = string(runes) + "…"
		if t.MinSize().Width <= maxW {
			return
		}
	}
}
