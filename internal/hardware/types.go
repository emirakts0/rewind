package hardware

import "fmt"

// FFmpegPath is the path to the FFmpeg executable.
var FFmpegPath = "bin/ffmpeg.exe"

// Vendor represents GPU vendor
type Vendor string

const (
	VendorNVIDIA  Vendor = "nvidia"
	VendorAMD     Vendor = "amd"
	VendorIntel   Vendor = "intel"
	VendorUnknown Vendor = "unknown"
)

// GPU represents a graphics processing unit
type GPU struct {
	Index    int
	Name     string
	Vendor   Vendor
	Encoders []Encoder
}

func (g *GPU) String() string {
	return fmt.Sprintf("[%d] %s (%s)", g.Index, g.Name, g.Vendor)
}

type GPUList []*GPU

func (l GPUList) FindByIndex(index int) *GPU {
	for _, g := range l {
		if g.Index == index {
			return g
		}
	}
	return nil
}

type Encoder struct {
	Name      string
	Codec     string
	Available bool
	GPUIndex  int // which GPU this encoder belongs to
}

type Display struct {
	Index        int
	Name         string
	FriendlyName string
	IsPrimary    bool
	Width        int
	Height       int
	X, Y         int
	GPUIndex     int
}

func (d *Display) String() string {
	primary := ""
	if d.IsPrimary {
		primary = " [primary]"
	}
	return fmt.Sprintf("[%d] %dx%d%s", d.Index, d.Width, d.Height, primary)
}

type DisplayList []*Display

func (l DisplayList) FindByIndex(index int) *Display {
	for _, d := range l {
		if d.Index == index {
			return d
		}
	}
	return nil
}

func (l DisplayList) FindPrimary() *Display {
	for _, d := range l {
		if d.IsPrimary {
			return d
		}
	}
	if len(l) > 0 {
		return l[0]
	}
	return nil
}

type SystemInfo struct {
	GPUs     GPUList
	Displays DisplayList
	Encoders []Encoder // all available encoders from all GPUs
}

// GetEncoder finds an encoder by name
func (s *SystemInfo) GetEncoder(name string) *Encoder {
	for i := range s.Encoders {
		if s.Encoders[i].Name == name {
			return &s.Encoders[i]
		}
	}
	return nil
}

// GetAvailableEncoders returns all available (working) encoders
func (s *SystemInfo) GetAvailableEncoders() []Encoder {
	var result []Encoder
	for _, e := range s.Encoders {
		if e.Available {
			result = append(result, e)
		}
	}
	return result
}

// GetDisplay finds a display by index
func (s *SystemInfo) GetDisplay(index int) *Display {
	return s.Displays.FindByIndex(index)
}
