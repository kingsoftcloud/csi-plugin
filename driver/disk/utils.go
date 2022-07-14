package driver

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

var (
	supportedAccessMode = &csi.VolumeCapability_AccessMode{
		Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
	}
)

const (
	// PublishInfoVolumeName is used to pass the volume name from
	// `ControllerPublishVolume` to `NodeStageVolume or `NodePublishVolume`
	// PublishInfoVolumeName = DriverName + "/volume-name"

	// minimumVolumeSizeInBytes is used to validate that the user is not trying
	// to create a volume that is smaller than what we support
	minimumVolumeSizeInBytes int64 = 1 * GB

	// maximumVolumeSizeInBytes is used to validate that the user is not trying
	// to create a volume that is larger than what we support
	maximumVolumeSizeInBytes int64 = 16 * TB

	// defaultVolumeSizeInBytes is used when the user did not provide a size or
	// the size they provided did not satisfy our requirements
	defaultVolumeSizeInBytes int64 = 16 * GB
)

func validateCapabilities(caps []*csi.VolumeCapability) bool {
	vcaps := []*csi.VolumeCapability_AccessMode{supportedAccessMode}

	hasSupport := func(mode csi.VolumeCapability_AccessMode_Mode) bool {
		for _, m := range vcaps {
			if mode == m.Mode {
				return true
			}
		}
		return false
	}

	supported := false
	for _, cap := range caps {
		if hasSupport(cap.AccessMode.Mode) {
			supported = true
		} else {
			// we need to make sure all capabilities are supported. Revert back
			// in case we have a cap that is supported, but is invalidated now
			supported = false
		}
	}

	return supported
}

// extractStorage extracts the storage size in bytes from the given capacity
// range. If the capacity range is not satisfied it returns the default volume
// size. If the capacity range is below or above supported sizes, it returns an
// error.
func extractStorage(capRange *csi.CapacityRange) (int64, error) {
	if capRange == nil {
		return defaultVolumeSizeInBytes, nil
	}

	requiredBytes := capRange.GetRequiredBytes()
	requiredSet := 0 < requiredBytes
	limitBytes := capRange.GetLimitBytes()
	limitSet := 0 < limitBytes

	if !requiredSet && !limitSet {
		return defaultVolumeSizeInBytes, nil
	}

	if requiredSet && limitSet && limitBytes < requiredBytes {
		return 0, fmt.Errorf("limit (%v) can not be less than required (%v) size", formatBytes(limitBytes), formatBytes(requiredBytes))
	}

	if requiredSet && !limitSet && requiredBytes < minimumVolumeSizeInBytes {
		return 0, fmt.Errorf("required (%v) can not be less than minimum supported volume size (%v)", formatBytes(requiredBytes), formatBytes(minimumVolumeSizeInBytes))
	}

	if limitSet && limitBytes < minimumVolumeSizeInBytes {
		return 0, fmt.Errorf("limit (%v) can not be less than minimum supported volume size (%v)", formatBytes(limitBytes), formatBytes(minimumVolumeSizeInBytes))
	}

	if requiredSet && requiredBytes > maximumVolumeSizeInBytes {
		return 0, fmt.Errorf("required (%v) can not exceed maximum supported volume size (%v)", formatBytes(requiredBytes), formatBytes(maximumVolumeSizeInBytes))
	}

	if !requiredSet && limitSet && limitBytes > maximumVolumeSizeInBytes {
		return 0, fmt.Errorf("limit (%v) can not exceed maximum supported volume size (%v)", formatBytes(limitBytes), formatBytes(maximumVolumeSizeInBytes))
	}

	if requiredSet && limitSet && requiredBytes == limitBytes {
		return requiredBytes, nil
	}

	if requiredSet {
		return requiredBytes, nil
	}

	if limitSet {
		return limitBytes, nil
	}

	return defaultVolumeSizeInBytes, nil
}

func formatBytes(inputBytes int64) string {
	output := float64(inputBytes)
	unit := ""

	switch {
	case inputBytes >= TB:
		output = output / TB
		unit = "Ti"
	case inputBytes >= GB:
		output = output / GB
		unit = "Gi"
	case inputBytes >= MB:
		output = output / MB
		unit = "Mi"
	case inputBytes >= KB:
		output = output / KB
		unit = "Ki"
	case inputBytes == 0:
		return "0"
	}

	result := strconv.FormatFloat(output, 'f', 1, 64)
	result = strings.TrimSuffix(result, ".0")
	return result + unit
}

type SuperMapString map[string]string

func (sm SuperMapString) Get(key, backup string) string {
	value, ok := sm[key]
	if !ok {
		return backup
	}
	return value
}
