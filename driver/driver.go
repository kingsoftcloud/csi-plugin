package driver

import (
	ebsClient "csi-plugin/pkg/ebs-client"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"

	"context"

	"google.golang.org/grpc"
	k8sclient "k8s.io/client-go/kubernetes"
)

type Driver struct {
	endpoint string
	srv      *grpc.Server
	readyMu  sync.Mutex
	ready    bool

	controllerServer csi.ControllerServer
	identityServer   csi.IdentityServer
	nodeServer       csi.NodeServer
}

type DriverConfig struct {
	ControllerServer bool
	NodeServer       bool
	EndPoint         string
	DriverName       string
	Version          string
	EbsClient        ebsClient.StorageService
	K8sclient        *k8sclient.Clientset
}

func NewDriver(config *DriverConfig) *Driver {
	if config.DriverName == "" {
		glog.Errorf("Driver name missing")
		return nil
	}
	// TODO version format and validation
	if len(config.Version) == 0 {
		glog.Errorf("Version argument missing")
		return nil
	}
	driver := &Driver{
		endpoint:         config.EndPoint,
		identityServer:   GetIdentityServer(config),
		controllerServer: nil,
		nodeServer:       nil,
		ready:            false,
	}
	if config.ControllerServer {
		driver.controllerServer = GetControllerServer(config)
	}
	if config.NodeServer {
		driver.nodeServer = GetNodeServer(config)
	}

	return driver
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

	csi.RegisterIdentityServer(d.srv, d.identityServer)
	if d.controllerServer != nil {
		csi.RegisterControllerServer(d.srv, d.controllerServer)
	}
	if d.nodeServer != nil {
		csi.RegisterNodeServer(d.srv, d.nodeServer)
	}

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
