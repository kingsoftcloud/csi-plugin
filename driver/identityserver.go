package driver

import (
	glog "github.com/Sirupsen/logrus"
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/golang/protobuf/ptypes/wrappers"
	"golang.org/x/net/context"
)

func (d *Driver) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	glog.Info("IdentityServer GetPluginInfo called...")
	resp := &csi.GetPluginInfoResponse{
		Name:          d.name,
		VendorVersion: d.version,
	}
	return resp, nil
}

func (d *Driver) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	glog.Info("IdentityServer GetPluginCapabilities called...")
	resp := &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			// {
			// 	Type: &csi.PluginCapability_Service_{
			// 		Service: &csi.PluginCapability_Service{
			// 			Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
			// 		},
			// 	},
			// },
			// {
			// 	Type: &csi.PluginCapability_Service_{
			// 		Service: &csi.PluginCapability_Service{
			// 			Type: csi.PluginCapability_Service_ACCESSIBILITY_CONSTRAINTS,
			// 		},
			// 	},
			// },
		},
	}
	return resp, nil
}

func (d *Driver) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	glog.Info("IdentityServer Probe called...")
	return &csi.ProbeResponse{Ready: &wrappers.BoolValue{Value: true}}, nil
}
