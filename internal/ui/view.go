// Package ui displays the tuner and routes user actions to the controller. It
// performs no audio capture and no pitch math.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// View is the main window and its widgets. User actions are surfaced as
// callbacks, which NewController wires to the controller.
type View struct {
	OnSelectTarget   func(name string)
	OnNext           func()
	OnStepTarget     func(delta int)
	OnDeviceSelected func(index int)
	OnRefreshDevices func()

	win       fyne.Window
	toolbar   *Toolbar
	hero      *NoteDisplay
	meter     *Meter
	directive *Directive
	kalimba   *KalimbaMap
}

// New builds the window within the given app.
func New(a fyne.App) *View {
	a.Settings().SetTheme(lightTheme{})
	loadTranslations()

	v := &View{}
	v.win = a.NewWindow("Kalimate")

	v.toolbar = NewToolbar(v.win.Canvas())
	v.toolbar.onSelect = func(i int) {
		if v.OnDeviceSelected != nil {
			v.OnDeviceSelected(i)
		}
	}
	v.toolbar.onRefresh = func() {
		if v.OnRefreshDevices != nil {
			v.OnRefreshDevices()
		}
	}

	v.hero = NewNoteDisplay()
	v.meter = NewMeter()
	v.directive = NewDirective()
	v.directive.OnNext = func() {
		if v.OnNext != nil {
			v.OnNext()
		}
	}
	v.kalimba = NewKalimbaMap()
	v.kalimba.OnTap = func(name string) {
		if v.OnSelectTarget != nil {
			v.OnSelectTarget(name)
		}
	}

	content := container.NewBorder(
		v.toolbar,
		v.kalimba,
		nil, nil,
		container.NewVBox(v.hero, v.meter, v.directive),
	)
	v.win.SetContent(content)
	v.win.Resize(fyne.NewSize(600, 760))
	v.win.SetFixedSize(true)

	// Keyboard shortcuts (no widget is focusable, so these reach the canvas):
	// ←/→ step the selected key along the instrument; Space/Enter advance.
	v.win.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		switch ev.Name {
		case fyne.KeyLeft:
			if v.OnStepTarget != nil {
				v.OnStepTarget(-1)
			}
		case fyne.KeyRight:
			if v.OnStepTarget != nil {
				v.OnStepTarget(1)
			}
		case fyne.KeySpace, fyne.KeyReturn, fyne.KeyEnter:
			if v.OnNext != nil {
				v.OnNext()
			}
		}
	})
	return v
}

// Window exposes the underlying window.
func (v *View) Window() fyne.Window { return v.win }

// SetDeviceList populates the toolbar's device menu and checkmark.
func (v *View) SetDeviceList(names []string, current int) {
	v.toolbar.SetDevices(names, current)
}

// Update renders one TunerState. Call on the UI goroutine (the controller wraps
// this in fyne.Do).
func (v *View) Update(s TunerState) {
	ph := phaseOf(s)
	z := zoneIdle
	if ph == phaseMatch {
		z = zoneOf(s.Cents, true)
	}

	v.toolbar.SetMic(s.MicName, s.MicError)
	v.toolbar.SetLevel(s.Level)

	v.hero.SetState(ph, s.Target, s.Heard, s.Cents, z)
	v.meter.SetState(ph == phaseMatch, s.Cents, z)

	next, allDone := nextPending(s.Target, func(n string) bool { return s.Status[n] == Done })
	targetDone := s.Target != "" && s.Status[s.Target] == Done
	v.directive.SetState(ph, s.Cents, z, s.Target, s.Heard, next, allDone, s.MicError, targetDone)

	heardTuned := s.Heard != "" && s.Heard == s.Target && ph == phaseMatch && z == zoneInTune
	v.kalimba.SetState(s.Status, s.Target, s.Heard, z, heardTuned)
}
