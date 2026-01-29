package audio

import (
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/gen2brain/malgo"
)

type DeviceType int

const (
	DeviceTypeInput DeviceType = iota
	DeviceTypeOutput
)

type Device struct {
	ID   string
	Name string
	Type DeviceType
}

func ListDevices() ([]Device, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to init audio context: %w", err)
	}
	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()

	var devices []Device

	captureInfos, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return nil, fmt.Errorf("failed to list capture devices: %w", err)
	}

	for _, info := range captureInfos {
		idHex := hex.EncodeToString(info.ID[:])
		name := info.Name()
		slog.Debug("found capture device", "name", name)
		devices = append(devices, Device{ID: idHex, Name: name, Type: DeviceTypeInput})
	}

	playbackInfos, err := ctx.Devices(malgo.Playback)
	if err != nil {
		return nil, fmt.Errorf("failed to list playback devices: %w", err)
	}

	for _, info := range playbackInfos {
		idHex := hex.EncodeToString(info.ID[:])
		name := info.Name()
		slog.Debug("found playback device", "name", name)
		devices = append(devices, Device{ID: idHex, Name: name, Type: DeviceTypeOutput})
	}

	return devices, nil
}

func ListInputDevices() ([]Device, error) {
	all, err := ListDevices()
	if err != nil {
		return nil, err
	}
	var result []Device
	for _, d := range all {
		if d.Type == DeviceTypeInput {
			result = append(result, d)
		}
	}
	return result, nil
}

func ListOutputDevices() ([]Device, error) {
	all, err := ListDevices()
	if err != nil {
		return nil, err
	}
	var result []Device
	for _, d := range all {
		if d.Type == DeviceTypeOutput {
			result = append(result, d)
		}
	}
	return result, nil
}

func ParseDeviceID(idHex string) (malgo.DeviceID, error) {
	bytes, err := hex.DecodeString(idHex)
	if err != nil {
		return malgo.DeviceID{}, err
	}
	var id malgo.DeviceID
	copy(id[:], bytes)
	return id, nil
}

func FindDeviceIDByName(name string) (string, error) {
	devices, err := ListDevices()
	if err != nil {
		return "", err
	}
	for _, d := range devices {
		if d.Name == name {
			return d.ID, nil
		}
	}
	return "", fmt.Errorf("device not found: %s", name)
}
