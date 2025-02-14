package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/longhorn/go-common-libs/types"
)

// GetKernelRelease returns the kernel release string.
func GetKernelRelease() (string, error) {
	utsname := &syscall.Utsname{}
	if err := syscall.Uname(utsname); err != nil {
		logrus.WithError(err).Warn("Failed to get kernel release")
		return "", err
	}

	// Extract the kernel release from the Utsname structure
	release := make([]byte, 0, len(utsname.Release))
	for _, b := range utsname.Release {
		if b == 0x00 {
			logrus.Trace("Found end of kernel release string [0x00]")
			break
		}
		release = append(release, byte(b))
	}
	return string(release), nil
}

// GetOSDistro reads the /etc/os-release file and returns the ID field.
func GetOSDistro(osReleaseContent string) (string, error) {
	var err error
	defer func() {
		err = errors.Wrapf(err, "failed to get host OS distro")
	}()

	if types.CachedOSDistro != "" {
		logrus.Tracef("Cached OS distro: %v", types.CachedOSDistro)
		return types.CachedOSDistro, nil
	}

	lines := strings.Split(osReleaseContent, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ID=") {
			id := strings.TrimPrefix(line, "ID=")
			id = strings.Trim(id, `"`)
			types.CachedOSDistro = id
			return GetOSDistro("")
		}
	}

	return "", fmt.Errorf("failed to find ID field in %v", types.OsReleaseFilePath)
}

// GetSystemBlockDeviceInfo returns the block device info for the system.
func GetSystemBlockDeviceInfo() (map[string]types.BlockDeviceInfo, error) {
	return getSystemBlockDeviceInfo(os.ReadDir, os.ReadFile)
}

// getSystemBlockDeviceInfo returns the block device info for the system.
// It injects the readDirFn and readFileFn for testing.
func getSystemBlockDeviceInfo(readDirFn func(string) ([]os.DirEntry, error), readFileFn func(string) ([]byte, error)) (map[string]types.BlockDeviceInfo, error) {
	devices, err := readDirFn(types.SysClassBlockDirectory)
	if err != nil {
		return nil, err
	}

	readDeviceNumber := func(numbers []string, index int) (int64, error) {
		if len(numbers) <= index {
			return 0, fmt.Errorf("invalid file format")
		}

		number, err := strconv.ParseInt(numbers[index], 10, 64)
		if err != nil {
			return 0, err
		}
		return number, nil
	}

	deviceInfo := make(map[string]types.BlockDeviceInfo, len(devices))
	for _, device := range devices {
		deviceName := device.Name()
		devicePath := filepath.Join(types.SysClassBlockDirectory, deviceName, "dev")

		data, err := readFileFn(devicePath)
		if err != nil {
			return nil, err
		}

		numbers := strings.Split(strings.TrimSpace(string(data)), ":")
		major, err := readDeviceNumber(numbers, 0)
		if err != nil {
			logrus.WithError(err).Warnf("failed to read device %s major", deviceName)
			continue
		}

		minor, err := readDeviceNumber(numbers, 1)
		if err != nil {
			logrus.WithError(err).Warnf("failed to read device %s minor", deviceName)
			continue
		}

		deviceInfo[deviceName] = types.BlockDeviceInfo{
			Name:  deviceName,
			Major: int(major),
			Minor: int(minor),
		}
	}
	return deviceInfo, nil
}
