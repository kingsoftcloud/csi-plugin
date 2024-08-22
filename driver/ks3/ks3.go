package ks3

import (
	csicommon "github.com/volcengine/volcengine-csi-driver/pkg/csi-common"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/klog/v2"
)

const (
	DefaultDriverName = "com.ksc.csi.ks3plugin"

	// Address of the ks3 server
	paramURL = "url"
	// Base directory of the ks3 to create volumes under.
	paramPath = "path"
	// Bucket of ks3
	paramBucket = "bucket"
	// Additional Args
	paramAdditionalArgs = "additional_args"
	// Debug level
	paramDbgLevel = "dbglevel"

	defaultDBGLevel          = "err"
	ks3PasswordFileDirectory = "/tmp/"
	socketPath               = "/tmp/ks3fs.sock"
	credentialID             = "akId"
	credentialKey            = "akSecret"

	// tempMntPath used for create ks3 sub directory
	tempMntPath = "/tmp/ks3_mnt/"
)

type Driver struct {
	*csicommon.CSIDriver
}

// NewDriver create the identity/node/controller server and disk driver
func NewDriver(name, version, nodeId string) *Driver {
	klog.Infof("Driver: %v version: %v", name, version)
	csiDriver := &csicommon.CSIDriver{}
	csiDriver.Name = DefaultDriverName
	if name != "" {
		csiDriver.Name = name
	}

	csiDriver.Version = version
	csiDriver.NodeID = nodeId
	csiDriver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
	})
	csiDriver.AddNodeServiceCapabilities([]csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_UNKNOWN,
	})

	return &Driver{
		CSIDriver: csiDriver,
	}
}

func (d *Driver) Run(endpoint string) {
	klog.Infof("Starting csi-plugin Driver: %v version: %v", d.Name, d.Version)

	s := csicommon.NewNonBlockingGRPCServer()

	s.Start(
		endpoint,
		NewIdentityServer(d),
		nil,
		NewNodeServer(d),
		false,
	)
	s.Wait()
}
