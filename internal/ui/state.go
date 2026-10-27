package ui

// TineStatus is the tuning progress of a key this session.
type TineStatus int

const (
	Pending TineStatus = iota // not yet confirmed in tune
	Done                      // confirmed in tune this session
)

// TunerState is the one struct the view renders per frame. The controller owns
// it; widgets never read detector internals.
type TunerState struct {
	Target    string                // selected key ("" = none selected)
	Heard     string                // kalimba key nearest the mic ("" = none)
	Cents     float64               // smoothed, relative to Target
	HasSignal bool                  // a pitch is currently being heard
	Level     float64               // 0..1 input level
	Status    map[string]TineStatus // per-key progress ("" map = nothing chosen yet)
	MicName   string                // current input device label
	MicError  bool                  // capture unavailable (permission/device gone)
}
