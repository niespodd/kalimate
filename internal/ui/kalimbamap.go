package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

const (
	kalTopPad    = 20
	kalSidePad   = 32
	kalBottomPad = 28
	kalCaptionH  = 16
	kalCapGap    = 8
	kalPadTop    = 12
	kalPadSide   = 14
	kalPadBot    = 16
	kalDotRowH   = 8
	kalDotGap    = 4
	kalLabelH    = 15
	kalLabelGap  = 6
	kalBridgeH   = 7
	kalTineArea  = 136
	kalTineW     = 9
)

// KalimbaMap is the primary control: 17 tines, a target selection, a "heard"
// dot, and per-key tuning status. Tapping a column selects that key as target.
type KalimbaMap struct {
	widget.BaseWidget
	// OnTap fires with a note name when its column is tapped.
	OnTap func(name string)

	status     map[string]TineStatus
	target     string
	heard      string
	zone       zoneKind // target's tuning zone
	heardTuned bool
	hover      int // key column under the mouse, or -1
}

func NewKalimbaMap() *KalimbaMap {
	k := &KalimbaMap{hover: -1}
	k.ExtendBaseWidget(k)
	return k
}

// colAt returns the key column under pos, or -1 if pos is outside the tappable
// body. Used by both tap routing and hover feedback so they agree exactly.
func (k *KalimbaMap) colAt(pos fyne.Position) int {
	size := k.Size()
	innerLeft := float32(kalSidePad + kalPadSide)
	innerW := size.Width - 2*(kalSidePad+kalPadSide)
	bodyTop := float32(kalTopPad + kalCaptionH + kalCapGap)
	bodyBottom := size.Height - kalBottomPad
	if pos.X < innerLeft || pos.X > innerLeft+innerW ||
		pos.Y < bodyTop || pos.Y > bodyBottom {
		return -1
	}
	col := int((pos.X - innerLeft) / (innerW / float32(len(Layout))))
	if col < 0 || col >= len(Layout) {
		return -1
	}
	return col
}

// MouseIn, MouseMoved, MouseOut and Cursor implement hover feedback: a pointer
// cursor and a faint highlight on the key column under the mouse.
func (k *KalimbaMap) MouseIn(ev *desktop.MouseEvent)    { k.setHover(k.colAt(ev.Position)) }
func (k *KalimbaMap) MouseMoved(ev *desktop.MouseEvent) { k.setHover(k.colAt(ev.Position)) }
func (k *KalimbaMap) MouseOut()                         { k.setHover(-1) }

func (k *KalimbaMap) setHover(col int) {
	if col != k.hover {
		k.hover = col
		k.Refresh()
	}
}

func (k *KalimbaMap) Cursor() desktop.Cursor {
	if k.hover >= 0 {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (k *KalimbaMap) SetState(status map[string]TineStatus, target, heard string, z zoneKind, heardTuned bool) {
	k.status, k.target, k.heard, k.zone, k.heardTuned = status, target, heard, z, heardTuned
	k.Refresh()
}

func (k *KalimbaMap) started() bool { return k.status != nil }

func (k *KalimbaMap) Tapped(ev *fyne.PointEvent) {
	if k.OnTap == nil {
		return
	}
	if col := k.colAt(ev.Position); col >= 0 {
		k.OnTap(Layout[col].Name)
	}
}

func (k *KalimbaMap) CreateRenderer() fyne.WidgetRenderer {
	r := &kalimbaRenderer{k: k}
	r.caption = newBoldText(tr("tapKeyToTune"), fyne.TextAlignLeading, 12)
	r.legend = newBoldText(tr("whatIHear"), fyne.TextAlignLeading, 12)
	r.legendDot = canvas.NewCircle(nil)
	r.count = newBoldText("", fyne.TextAlignTrailing, 12)
	r.body = canvas.NewRectangle(nil)
	r.body.CornerRadius = 16
	r.bridge = canvas.NewRectangle(nil)
	r.bridge.CornerRadius = 2
	r.ring = canvas.NewRectangle(color.Transparent)
	r.ring.StrokeWidth = 2.5
	r.ring.CornerRadius = 6.5
	r.chip = canvas.NewRectangle(nil)
	r.chip.CornerRadius = 4
	r.hoverChip = canvas.NewRectangle(nil)
	r.hoverChip.CornerRadius = 4
	r.kbdLeft = newKbdChip("←")
	r.kbdRight = newKbdChip("→")
	r.heardDot = canvas.NewCircle(nil)
	for range Layout {
		lbl := canvas.NewText("", nil)
		lbl.TextSize = 11
		lbl.Alignment = fyne.TextAlignCenter
		r.labels = append(r.labels, lbl)
		tine := canvas.NewRectangle(nil)
		tine.CornerRadius = 4
		r.tines = append(r.tines, tine)
	}
	return r
}

type kalimbaRenderer struct {
	k         *KalimbaMap
	caption   *canvas.Text
	legend    *canvas.Text
	legendDot *canvas.Circle
	count     *canvas.Text
	body      *canvas.Rectangle
	bridge    *canvas.Rectangle
	ring      *canvas.Rectangle
	chip      *canvas.Rectangle
	hoverChip *canvas.Rectangle
	kbdLeft   *kbdChip
	kbdRight  *kbdChip
	heardDot  *canvas.Circle
	labels    []*canvas.Text
	tines     []*canvas.Rectangle
}

func (r *kalimbaRenderer) Layout(size fyne.Size) {
	contentLeft := float32(kalSidePad)
	contentW := size.Width - 2*kalSidePad

	r.caption.Move(fyne.NewPos(contentLeft, kalTopPad))
	r.caption.Resize(fyne.NewSize(contentW, kalCaptionH))

	// Right caption cluster: [dot] what I hear  ·  N of 17 tuned, right-aligned.
	countW := r.count.MinSize().Width
	legendW := r.legend.MinSize().Width
	right := contentLeft + contentW
	r.count.Move(fyne.NewPos(right-countW, kalTopPad))
	r.count.Resize(fyne.NewSize(countW, kalCaptionH))
	legendX := right - countW - 14 - legendW
	r.legend.Move(fyne.NewPos(legendX, kalTopPad))
	r.legend.Resize(fyne.NewSize(legendW, kalCaptionH))
	dcy := float32(kalTopPad) + kalCaptionH/2
	r.legendDot.Position1 = fyne.NewPos(legendX-6-7, dcy-3.5)
	r.legendDot.Position2 = fyne.NewPos(legendX-6, dcy+3.5)

	// Arrow-key hint, just right of the caption; hidden if it would collide with
	// the "what I hear" cluster on a narrow window.
	capRight := contentLeft + r.caption.MinSize().Width + 12
	if capRight+r.kbdLeft.width()+3+r.kbdRight.width() <= legendX-20 {
		r.kbdLeft.show()
		r.kbdRight.show()
		r.kbdLeft.layout(capRight, dcy)
		r.kbdRight.layout(capRight+r.kbdLeft.width()+3, dcy)
	} else {
		r.kbdLeft.hide()
		r.kbdRight.hide()
	}

	bodyTop := float32(kalTopPad + kalCaptionH + kalCapGap)
	bodyBottom := size.Height - kalBottomPad
	r.body.Move(fyne.NewPos(contentLeft, bodyTop))
	r.body.Resize(fyne.NewSize(contentW, bodyBottom-bodyTop))

	innerLeft := contentLeft + kalPadSide
	innerW := contentW - 2*kalPadSide
	innerTop := bodyTop + kalPadTop
	slot := innerW / float32(len(Layout))

	labelsY := innerTop + kalDotRowH + kalDotGap
	bridgeY := labelsY + kalLabelH + kalLabelGap
	r.bridge.Move(fyne.NewPos(innerLeft, bridgeY))
	r.bridge.Resize(fyne.NewSize(innerW, kalBridgeH))
	tineTop := bridgeY + kalBridgeH

	for i, t := range Layout {
		cx := innerLeft + slot*(float32(i)+0.5)
		r.labels[i].Move(fyne.NewPos(cx-slot/2, labelsY))
		r.labels[i].Resize(fyne.NewSize(slot, kalLabelH))

		length := float32(132 - (t.Midi-60)*29/10)
		r.tines[i].Move(fyne.NewPos(cx-kalTineW/2, tineTop))
		r.tines[i].Resize(fyne.NewSize(kalTineW, length))

		if t.Name == r.k.target {
			r.ring.Move(fyne.NewPos(cx-kalTineW/2-5, tineTop-5))
			r.ring.Resize(fyne.NewSize(kalTineW+10, length+10))
			// chip behind the target label
			lw := r.labels[i].MinSize().Width
			r.chip.Move(fyne.NewPos(cx-lw/2-4, labelsY-1))
			r.chip.Resize(fyne.NewSize(lw+8, kalLabelH+2))
		}
		if t.Name == r.k.heard {
			hcx := cx
			r.heardDot.Position1 = fyne.NewPos(hcx-3.5, innerTop)
			r.heardDot.Position2 = fyne.NewPos(hcx+3.5, innerTop+7)
		}
	}

	if r.k.hover >= 0 {
		cx := innerLeft + slot*(float32(r.k.hover)+0.5)
		lw := r.labels[r.k.hover].MinSize().Width
		r.hoverChip.Move(fyne.NewPos(cx-lw/2-4, labelsY-1))
		r.hoverChip.Resize(fyne.NewSize(lw+8, kalLabelH+2))
	}
}

func (r *kalimbaRenderer) Refresh() {
	pal := currentPalette()
	r.caption.Color = pal.kalCaption
	r.legend.Color = pal.kalCaption
	r.legendDot.FillColor = pal.secondary
	r.count.Color = pal.kalCaption
	r.body.FillColor = pal.kalBody
	r.bridge.FillColor = pal.kalBridge

	started := r.k.started()
	done := 0
	for _, t := range Layout {
		if r.k.status[t.Name] == Done {
			done++
		}
	}
	r.count.Text = tr("tunedCount", map[string]any{"Done": done, "Total": len(Layout)})

	for i, t := range Layout {
		lbl := r.labels[i]
		lbl.Text = t.Name
		tine := r.tines[i]
		switch {
		case !started:
			tine.FillColor = pal.kalNeutral
			lbl.Color = pal.lblPending
			lbl.TextStyle = fyne.TextStyle{}
		case t.Name == r.k.target:
			tine.FillColor = pal.targetColor(r.k.zone)
			lbl.Color = pal.lblTarget
			lbl.TextStyle = fyne.TextStyle{Bold: true}
		case r.k.status[t.Name] == Done:
			tine.FillColor = pal.kalDone
			lbl.Color = pal.lblDone
			lbl.TextStyle = fyne.TextStyle{}
		default:
			tine.FillColor = pal.kalPending
			lbl.Color = pal.lblPending
			lbl.TextStyle = fyne.TextStyle{}
		}
	}

	// Target ring + label chip.
	if started && r.k.target != "" {
		r.ring.Show()
		r.ring.StrokeColor = pal.ring
		r.chip.Show()
		r.chip.FillColor = withAlpha(color.White, 0.75)
	} else {
		r.ring.Hide()
		r.chip.Hide()
	}

	// Hover highlight: a faint chip behind the hovered key's label. Skip the
	// current target, which already carries its own chip.
	if r.k.hover >= 0 && Layout[r.k.hover].Name != r.k.target {
		r.hoverChip.Show()
		r.hoverChip.FillColor = pal.kalHover
	} else {
		r.hoverChip.Hide()
	}

	r.kbdLeft.applyColors(pal)
	r.kbdRight.applyColors(pal)

	// Heard dot.
	if r.k.heard != "" {
		r.heardDot.Show()
		if r.k.heardTuned {
			r.heardDot.FillColor = pal.heardTuned
		} else {
			r.heardDot.FillColor = pal.heardOff
		}
	} else {
		r.heardDot.Hide()
	}

	r.Layout(r.k.Size())
	canvas.Refresh(r.k)
}

func (r *kalimbaRenderer) MinSize() fyne.Size {
	h := kalTopPad + kalCaptionH + kalCapGap +
		(kalPadTop + kalDotRowH + kalDotGap + kalLabelH + kalLabelGap + kalBridgeH + kalTineArea + kalPadBot) +
		kalBottomPad
	return fyne.NewSize(460, float32(h))
}

func (r *kalimbaRenderer) Objects() []fyne.CanvasObject {
	objs := []fyne.CanvasObject{r.body, r.ring}
	for _, t := range r.tines {
		objs = append(objs, t)
	}
	objs = append(objs, r.bridge, r.chip, r.hoverChip)
	for _, l := range r.labels {
		objs = append(objs, l)
	}
	objs = append(objs, r.heardDot, r.caption, r.legend, r.legendDot, r.count)
	objs = append(objs, r.kbdLeft.objects()...)
	return append(objs, r.kbdRight.objects()...)
}

func (r *kalimbaRenderer) Destroy() {}
