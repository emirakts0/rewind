package gpu

import "fmt"

type Vendor string

const (
	VendorNVIDIA  Vendor = "nvidia"
	VendorAMD     Vendor = "amd"
	VendorIntel   Vendor = "intel"
	VendorUnknown Vendor = "unknown"
)

type GPU struct {
	Index        int
	Name         string
	Vendor       Vendor
	VRAM         uint64
	IsIntegrated bool
	LUID         uint64
	Encoders     []Encoder
}

type Encoder struct {
	Name      string
	Codec     string
	Available bool
}

func (g *GPU) String() string {
	gpuType := "discrete"
	if g.IsIntegrated {
		gpuType = "integrated"
	}
	return fmt.Sprintf("[%d] %s (%s, %s)", g.Index, g.Name, g.Vendor, gpuType)
}

func (g *GPU) GetPreferredEncoder() *Encoder {
	for i := range g.Encoders {
		if g.Encoders[i].Available && g.Encoders[i].Codec == "h264" {
			return &g.Encoders[i]
		}
	}
	for i := range g.Encoders {
		if g.Encoders[i].Available && g.Encoders[i].Codec == "hevc" {
			return &g.Encoders[i]
		}
	}
	return nil
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

func (l GPUList) HasHybridSetup() bool {
	hasIntegrated := false
	hasDiscrete := false
	for _, g := range l {
		if g.IsIntegrated {
			hasIntegrated = true
		} else {
			hasDiscrete = true
		}
	}
	return hasIntegrated && hasDiscrete
}
