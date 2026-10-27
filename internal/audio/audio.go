// Package audio interfaces with capture devices. It knows nothing about
// pitch or display: it enumerates microphones and streams raw mono samples.
package audio

import (
	"encoding/binary"
	"math"
	"sync/atomic"

	"github.com/gen2brain/malgo"
)

// Device is a selectable capture input.
type Device struct {
	Name      string
	IsDefault bool
	id        malgo.DeviceID
}

// ListDevices returns the available capture (input) devices.
func ListDevices() ([]Device, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = ctx.Uninit(); ctx.Free() }()

	infos, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return nil, err
	}

	devices := make([]Device, 0, len(infos))
	for _, info := range infos {
		devices = append(devices, Device{
			Name:      info.Name(),
			IsDefault: info.IsDefault != 0,
			id:        info.ID,
		})
	}
	return devices, nil
}

// Capture streams fixed-size frames of mono f32 samples from one device.
type Capture struct {
	// Frames delivers analysis windows of frameSize samples each. A frame is
	// dropped if the consumer is not keeping up — freshness beats completeness.
	Frames chan []float64

	ctx       *malgo.AllocatedContext
	dev       *malgo.Device
	pending   []float64
	frameSize int
	dropped   uint32 // frames discarded because the consumer fell behind
}

// Dropped reports the running count of frames dropped on overrun. A change in
// this value between two received frames means the stream is discontinuous, so
// the consumer must not treat the frames on either side as adjacent.
func (c *Capture) Dropped() uint32 { return atomic.LoadUint32(&c.dropped) }

// Open starts capturing from the given device (nil = system default).
func Open(dev *Device, sampleRate, frameSize int) (*Capture, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}

	c := &Capture{
		Frames:    make(chan []float64, 4),
		ctx:       ctx,
		pending:   make([]float64, 0, frameSize*2),
		frameSize: frameSize,
	}

	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.Capture.Format = malgo.FormatF32
	cfg.Capture.Channels = 1
	cfg.SampleRate = uint32(sampleRate)
	cfg.Alsa.NoMMap = 1
	if dev != nil {
		id := dev.id
		cfg.Capture.DeviceID = id.Pointer()
	}

	device, err := malgo.InitDevice(ctx.Context, cfg, malgo.DeviceCallbacks{Data: c.onData})
	if err != nil {
		_ = ctx.Uninit()
		ctx.Free()
		return nil, err
	}
	c.dev = device

	if err := device.Start(); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

// onData runs on the audio thread: accumulate samples, emit whole frames.
func (c *Capture) onData(_, in []byte, frames uint32) {
	for i := 0; i < int(frames); i++ {
		bits := binary.LittleEndian.Uint32(in[i*4:])
		c.pending = append(c.pending, float64(math.Float32frombits(bits)))
	}

	for len(c.pending) >= c.frameSize {
		frame := make([]float64, c.frameSize)
		copy(frame, c.pending[:c.frameSize])
		c.pending = c.pending[c.frameSize:]

		select {
		case c.Frames <- frame:
		default:
			atomic.AddUint32(&c.dropped, 1)
		}
	}
}

// Close stops capture and releases the device.
func (c *Capture) Close() {
	if c.dev != nil {
		c.dev.Uninit()
		c.dev = nil
	}
	if c.ctx != nil {
		_ = c.ctx.Uninit()
		c.ctx.Free()
		c.ctx = nil
	}
}
