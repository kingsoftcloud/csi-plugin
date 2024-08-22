package ks3

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// IdentityServer driver
type IdentityServer struct {
	d *Driver
}

// NewIdentityServer new IdentityServer
func NewIdentityServer(driver *Driver) *IdentityServer {
	return &IdentityServer{driver}
}

// GetPluginInfo return info of the plugin
func (ids *IdentityServer) GetPluginInfo(_ context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	if ids.d.Name == "" {
		return nil, status.Error(codes.Unavailable, "Plugin name is not configured")
	}
	if ids.d.Version == "" {
		return nil, status.Error(codes.Unavailable, "Plugin version is not configured")
	}

	resp := &csi.GetPluginInfoResponse{
		Name:          ids.d.Name,
		VendorVersion: ids.d.Version,
	}

	return resp, nil
}

// Probe check whether the plugin is running or not.
func (ids *IdentityServer) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{Ready: &wrapperspb.BoolValue{Value: true}}, nil
}

// GetPluginCapabilities return the capabilities of the plugin
func (ids *IdentityServer) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_UNKNOWN,
					},
				},
			},
		},
	}, nil
}
