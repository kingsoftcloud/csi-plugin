package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	k8svol "k8s.io/kubernetes/pkg/volume"
	"k8s.io/kubernetes/pkg/volume/util/fs"
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
			klog.Fatalf("Couldn't determine hostname: %v", err)
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
	id, err := ioutil.ReadFile(PYFILE)
	if err != nil {
		// 兼容客户EPC节点无此文件
		if productuuid, err := ioutil.ReadFile(path.Join(dmiDir, "id", "product_uuid")); err != nil {
			return "", fmt.Errorf("not found physical system uuid from %s and %s, err: %v", dmiDir, PYFILE, err)
		} else {
			return strings.ToLower(strings.TrimSpace(string(productuuid))), nil
		}
	}
	return strings.ToLower(strings.TrimSpace(string(id))), nil
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

// GetMetrics get path metric
func GetMetrics(path string) (*csi.NodeGetVolumeStatsResponse, error) {
	if path == "" {
		return nil, fmt.Errorf("getMetrics No path given")
	}
	available, capacity, usage, inodes, inodesFree, inodesUsed, err := fs.FsInfo(path)
	if err != nil {
		return nil, err
	}

	metrics := &k8svol.Metrics{Time: metav1.Now()}
	metrics.Available = resource.NewQuantity(available, resource.BinarySI)
	metrics.Capacity = resource.NewQuantity(capacity, resource.BinarySI)
	metrics.Used = resource.NewQuantity(usage, resource.BinarySI)
	metrics.Inodes = resource.NewQuantity(inodes, resource.BinarySI)
	metrics.InodesFree = resource.NewQuantity(inodesFree, resource.BinarySI)
	metrics.InodesUsed = resource.NewQuantity(inodesUsed, resource.BinarySI)

	metricAvailable, ok := (*(metrics.Available)).AsInt64()
	if !ok {
		klog.Errorf("failed to fetch available bytes for target: %s", path)
		return nil, status.Error(codes.Unknown, "failed to fetch available bytes")
	}
	metricCapacity, ok := (*(metrics.Capacity)).AsInt64()
	if !ok {
		klog.Errorf("failed to fetch capacity bytes for target: %s", path)
		return nil, status.Error(codes.Unknown, "failed to fetch capacity bytes")
	}
	metricUsed, ok := (*(metrics.Used)).AsInt64()
	if !ok {
		klog.Errorf("failed to fetch used bytes for target %s", path)
		return nil, status.Error(codes.Unknown, "failed to fetch used bytes")
	}
	metricInodes, ok := (*(metrics.Inodes)).AsInt64()
	if !ok {
		klog.Errorf("failed to fetch available inodes for target %s", path)
		return nil, status.Error(codes.Unknown, "failed to fetch available inodes")
	}
	metricInodesFree, ok := (*(metrics.InodesFree)).AsInt64()
	if !ok {
		klog.Errorf("failed to fetch free inodes for target: %s", path)
		return nil, status.Error(codes.Unknown, "failed to fetch free inodes")
	}
	metricInodesUsed, ok := (*(metrics.InodesUsed)).AsInt64()
	if !ok {
		klog.Errorf("failed to fetch used inodes for target: %s", path)
		return nil, status.Error(codes.Unknown, "failed to fetch used inodes")
	}

	return &csi.NodeGetVolumeStatsResponse{
		Usage: []*csi.VolumeUsage{
			{
				Available: metricAvailable,
				Total:     metricCapacity,
				Used:      metricUsed,
				Unit:      csi.VolumeUsage_BYTES,
			},
			{
				Available: metricInodesFree,
				Total:     metricInodes,
				Used:      metricInodesUsed,
				Unit:      csi.VolumeUsage_INODES,
			},
		},
	}, nil
}
