package ui

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/software"
	"fyne.io/fyne/v2/test"
)

func statusWith(done ...string) map[string]TineStatus {
	m := make(map[string]TineStatus, len(Layout))
	for _, t := range Layout {
		m[t.Name] = Pending
	}
	for _, n := range done {
		m[n] = Done
	}
	return m
}

const mic = "MacBook Pro Microphone"

var fixtures = []struct {
	name  string
	state TunerState
}{
	{"idle", TunerState{MicName: mic}},
	{"sharp", TunerState{
		Target: "C4", Heard: "C4", Cents: 18, HasSignal: true, Level: 0.6,
		Status: statusWith("E6", "C6", "A5", "F5", "D5", "B4"), MicName: mic,
	}},
	{"wrongkey", TunerState{
		Target: "C4", Heard: "D4", HasSignal: true, Level: 0.5,
		Status: statusWith("E6", "C6", "A5", "F5", "D5", "B4"), MicName: mic,
	}},
	{"intune", TunerState{
		Target: "C4", Heard: "C4", Cents: 1, HasSignal: true, Level: 0.8,
		Status: statusWith("E6", "C6", "A5", "F5", "D5", "B4", "G4", "E4"), MicName: mic,
	}},
	{"nosignal", TunerState{
		Target: "G4", HasSignal: false, Level: 0,
		Status: statusWith("E6", "C6"), MicName: mic,
	}},
}

// TestRenderFixtures renders the full view for each fixture to /tmp/kalimate-ui/.
func TestRenderFixtures(t *testing.T) {
	dir := "/tmp/kalimate-ui"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	app := test.NewApp()
	defer test.NewApp()

	for _, f := range fixtures {
		v := New(app)
		v.SetDeviceList([]string{mic}, 0)
		v.Update(f.state)

		w := v.Window()
		w.SetFixedSize(false)
		w.Resize(fyne.NewSize(600, 760))

		img := software.RenderCanvas(w.Canvas(), lightTheme{})
		writePNG(t, filepath.Join(dir, "light-"+f.name+".png"), img)
	}
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	fh, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()
	if err := png.Encode(fh, img); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s (%dx%d)", path, img.Bounds().Dx(), img.Bounds().Dy())
}
