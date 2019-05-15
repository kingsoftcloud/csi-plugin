package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/golang/glog"
)

const (
	blockDir     = "/sys/block"
	cacheDir     = "/sys/devices/system/cpu/cpu"
	netDir       = "/sys/class/net"
	dmiDir       = "/sys/class/dmi"
	ppcDevTree   = "/proc/device-tree"
	s390xDevTree = "/etc" // s390/s390x changes

	NodeRegionKey = "failure-domain.beta.kubernetes.io/region"
	NodeZoneKey   = "failure-domain.beta.kubernetes.io/zone"
	NodeRoleKey   = "kubernetes.io/role"
)

var (
	PYFILE = "/etc/.instance_id"
	VMS    = []string{"KVM", "kvm", "XEN", "xen", "VirtualBox", "virtualbox"}
)

// GetHostname returns OS's hostname if 'hostnameOverride' is empty; otherwise, return 'hostnameOverride'.
func GetHostname(hostnameOverride string) string {
	hostname := hostnameOverride
	if hostname == "" {
		nodename, err := os.Hostname()
		if err != nil {
			glog.Fatalf("Couldn't determine hostname: %v", err)
		}
		hostname = nodename
	}
	return strings.ToLower(strings.TrimSpace(hostname))
}

func GetSystemUUID() (string, error) {
	ok, err := IsPhysical()
	if err != nil {
		return "", err
	}

	if ok {
		return GetPYUUID()
	} else {
		return GetVMUUID()
	}
}

func GetVMUUID() (string, error) {
	if id, err := ioutil.ReadFile(path.Join(dmiDir, "id", "product_uuid")); err == nil {
		return strings.ToLower(strings.TrimSpace(string(id))), nil
	} else if id, err = ioutil.ReadFile(path.Join(ppcDevTree, "system-id")); err == nil {
		return strings.ToLower(strings.TrimSpace(string(id))), nil
	} else if id, err = ioutil.ReadFile(path.Join(ppcDevTree, "vm,uuid")); err == nil {
		return strings.ToLower(strings.TrimSpace(string(id))), nil
	} else if id, err = ioutil.ReadFile(path.Join(s390xDevTree, "machine-id")); err == nil {
		return strings.ToLower(strings.TrimSpace(string(id))), nil
	} else {
		return "", err
	}
}

func GetPYUUID() (string, error) {
	if id, err := ioutil.ReadFile(PYFILE); err == nil {
		return strings.ToLower(strings.TrimSpace(string(id))), nil
	}

	return "", fmt.Errorf("not found physical system uuid from %s", PYFILE)
}

func IsPhysical() (bool, error) {
	vmName, err := ioutil.ReadFile(path.Join(dmiDir, "id", "product_name"))
	if err != nil {
		return false, fmt.Errorf("not found file %s", path.Join(dmiDir, "id", "product_name"))
	}

	for _, v := range VMS {
		if strings.ToLower(strings.TrimSpace(string(vmName))) == v {
			return false, nil
		}
	}

	return true, nil
}
