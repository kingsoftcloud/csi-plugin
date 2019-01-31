package driver

import (
	ebsClient "csi-plugin/pkg/ebs-client"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/golang/glog"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"

	"context"

	"google.golang.org/grpc"
	k8sclient "k8s.io/client-go/kubernetes"
)

type Driver struct {
	name     string
	nodeID   string
	version  string
	endpoint string
	region   string

	ebsClient ebsClient.StorageService
	k8sclient *k8sclient.Clientset
	srv       *grpc.Server

	readyMu sync.Mutex
	ready   bool
}

type DriverConfig struct {
	EndPoint   string
	DriverName string
	NodeID     string
	Version    string
	Region     string
}

func NewDriver(config *DriverConfig, ebsClient ebsClient.StorageService, k8sclient *k8sclient.Clientset) *Driver {
	if config.DriverName == "" {
		glog.Errorf("Driver name missing")
		return nil
	}

	if config.NodeID == "" {
		glog.Errorf("NodeID missing")
		return nil
	}
	// TODO version format and validation
	if len(config.Version) == 0 {
		glog.Errorf("Version argument missing")
		return nil
	}

	return &Driver{
		name:      config.DriverName,
		nodeID:    config.NodeID,
		version:   config.Version,
		endpoint:  config.EndPoint,
		region:    config.Region,
		ebsClient: ebsClient,
		k8sclient: k8sclient,
	}
}

func (d *Driver) Run() error {
	proto, addr, err := ParseEndpoint(d.endpoint)
	if err != nil {
		glog.Fatal(err.Error())
	}

	if proto == "unix" {
		addr = "/" + addr
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			glog.Fatalf("Failed to remove %s, error: %s", addr, err.Error())
		}
	}

	listener, err := net.Listen(proto, addr)
	if err != nil {
		glog.Fatalf("Failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logGRPC),
	}
	d.srv = grpc.NewServer(opts...)

	csi.RegisterIdentityServer(d.srv, d)
	csi.RegisterControllerServer(d.srv, d)
	csi.RegisterNodeServer(d.srv, d)

	d.ready = true
	glog.Infof("Listening for connections on address: %#v", listener.Addr())

	return d.srv.Serve(listener)
}

func (d *Driver) Stop() {
	d.readyMu.Lock()
	d.ready = false
	d.readyMu.Unlock()

	glog.Info("server stopped")
	d.srv.GracefulStop()
}

func (d *Driver) ForceStop() {
	d.srv.Stop()
}

func ParseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("Invalid endpoint: %v", ep)
}

func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	fmt.Println()
	glog.V(3).Infof("GRPC call: %s", info.FullMethod)
	glog.V(5).Infof("GRPC request: %s", protosanitizer.StripSecretsCSI03(req))
	resp, err := handler(ctx, req)
	if err != nil {
		glog.Errorf("GRPC error: %v", err)
	} else {
		glog.V(5).Infof("GRPC response: %s", protosanitizer.StripSecretsCSI03(resp))
	}
	return resp, err
}
