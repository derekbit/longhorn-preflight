package namespace

import (
	"github.com/sirupsen/logrus"

	"github.com/longhorn/go-common-libs/types"
)

func GetDefaultProcessName() string {
	osDistro, err := GetOSDistro()
	if err != nil {
		logrus.Trace("failed to get os distro, fallback to default host process")
		return types.ProcessNone
	}

	switch osDistro {
	case types.OSDistroTalosLinux:
		return types.ProcessKubelet
	default:
		return types.ProcessNone
	}
}
