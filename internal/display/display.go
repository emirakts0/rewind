package display

import "fmt"

type Display struct {
	Index        int
	Name         string
	FriendlyName string
	IsPrimary    bool
	Width        int
	Height       int
	RefreshRate  int
	X, Y         int
	GPUIndex     int
	GPUName      string
}

func (d *Display) String() string {
	primary := ""
	if d.IsPrimary {
		primary = " [primary]"
	}
	return fmt.Sprintf("[%d] %dx%d%s -> %s", d.Index, d.Width, d.Height, primary, d.GPUName)
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
