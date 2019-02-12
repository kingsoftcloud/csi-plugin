/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package node

import (
	"github.com/golang/glog"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"fmt"
)

const (
	blockDir     = "/sys/block"
	cacheDir     = "/sys/devices/system/cpu/cpu"
	netDir       = "/sys/class/net"
	dmiDir       = "/sys/class/dmi"
	ppcDevTree   = "/proc/device-tree"
	s390xDevTree = "/etc" // s390/s390x changes
)

var (
	PYFILE = "/etc/.instance_id"
	VMS = []string{"KVM", "kvm", "XEN", "xen", "VirtualBox", "virtualbox"}
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