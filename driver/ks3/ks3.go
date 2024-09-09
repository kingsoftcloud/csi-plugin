package ks3

import (
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
	Name    string
	NodeID  string
	Version string
	Cap     []*csi.VolumeCapability_AccessMode
	CSCap   []*csi.ControllerServiceCapability
	NSCap   []*csi.NodeServiceCapability
}

// NewDriver create the identity/node/controller server and disk driver
func NewDriver(name, version, nodeId string) *Driver {
	klog.Infof("Driver: %v version: %v", name, version)
	csiDriver := &Driver{}
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

	return csiDriver
}

func (d *Driver) Run(endpoint string) {
	klog.Infof("Starting csi-plugin Driver: %v version: %v", d.Name, d.Version)

	s := NewNonBlockingGRPCServer()

	s.Start(
		endpoint,
		NewIdentityServer(d),
		nil,
		NewNodeServer(d),
		false,
	)
	s.Wait()
}

// AddVolumeCapabilityAccessModes add VolumeCapability_AccessMode_Mode
func (d *Driver) AddVolumeCapabilityAccessModes(vc []csi.VolumeCapability_AccessMode_Mode) []*csi.VolumeCapability_AccessMode {
	var vca []*csi.VolumeCapability_AccessMode
	for _, c := range vc {
		klog.Infof("Enabling volume access mode: %v", c.String())
		vca = append(vca, &csi.VolumeCapability_AccessMode{Mode: c})
	}
	d.Cap = vca
	return vca
}

// AddNodeServiceCapabilities add NodeServiceCapability_RPC_Type
func (d *Driver) AddNodeServiceCapabilities(nl []csi.NodeServiceCapability_RPC_Type) {
	var nsc []*csi.NodeServiceCapability
	for _, n := range nl {
		klog.Infof("Enabling node service capability: %v", n.String())
		nsc = append(nsc, NewNodeServiceCapability(n))
	}
	d.NSCap = nsc
}
