package driver

import (
	ebsClient "csi-plugin/pkg/ebs-client"
	api "csi-plugin/pkg/open-api"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/klog/v2"

	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"

	"context"

	snapClientset "github.com/kubernetes-csi/external-snapshotter/client/v4/clientset/versioned"
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

type Config struct {
	Version                string
	EndPoint               string
	DriverName             string
	EnableNodeServer       bool
	EnableControllerServer bool
	EnableVolumeExpansion  bool
	MaxVolumeSize          int64
	EbsClient              ebsClient.StorageService
	K8sClient              *k8sclient.Clientset
	MetricEnabled          bool
	MaxVolumesPerNode      int64
}

// GlobalConfig save global values for plugin
type GlobalConfig struct {
	K8sClient     *k8sclient.Clientset
	EbsClient     ebsClient.StorageService
	OpenApiConfig *api.ClientConfig
	SnapClient    *snapClientset.Clientset
}

var (
	GlobalConfigVar GlobalConfig
)

func NewDriver(config *Config) *Driver {
	if config.DriverName == "" {
		klog.Errorf("Driver name missing")
		return nil
	}
	// TODO version format and validation
	if len(config.Version) == 0 {
		klog.Errorf("Version argument missing")
		return nil
	}
	driver := &Driver{
		endpoint:       config.EndPoint,
		identityServer: GetIdentityServer(config),
		ready:          false,
	}
	if config.EnableControllerServer {
		driver.controllerServer = GetControllerServer(config)
	}
	if config.EnableNodeServer {
		driver.nodeServer = GetNodeServer(config)
	}

	return driver
}

func (d *Driver) Run() error {
	proto, addr, err := ParseEndpoint(d.endpoint)
	if err != nil {
		klog.Fatal(err.Error())
		return err
	}

	if proto == "unix" {
		addr = "/" + addr
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			klog.Fatalf("Failed to remove %s, error: %s", addr, err.Error())
			return err
		}
	}

	listener, err := net.Listen(proto, addr)
	if err != nil {
		klog.Fatalf("Failed to listen: %v", err)
		return err
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

	klog.V(2).Infof("Listening for connections on address: %#v", listener.Addr())
	return d.srv.Serve(listener)
}

func (d *Driver) Stop() {
	d.readyMu.Lock()
	d.ready = false
	d.readyMu.Unlock()

	klog.V(2).Info("server stopped")
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
	return "", "", fmt.Errorf("invalid endpoint: %v", ep)
}

func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	level := klog.Level(getLogLevel(info.FullMethod))

	if ShoudLog(info.FullMethod) {
		klog.V(level).Infof("GRPC call: %s", info.FullMethod)
		klog.V(level).Infof("GRPC request: %s", protosanitizer.StripSecrets(req))
	}
	resp, err := handler(ctx, req)
	if err != nil && !strings.Contains(err.Error(), "Repeatedly sending the same creation request within a short period of time") {
		klog.Errorf("GRPC error: %v", err)
	} else if ShoudLog(info.FullMethod) {
		klog.V(level).Infof("GRPC response: %s", protosanitizer.StripSecrets(resp))
	}
	return resp, err
}
