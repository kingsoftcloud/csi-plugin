package driver

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	ebsClient "csi-plugin/pkg/ebs-client"

	"github.com/kubernetes-csi/csi-test/pkg/sanity"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	flag.Set("logtostderr", "true")
	flag.Set("v", "5")
}

var (
	socket     = "/tmp/csi.sock"
	endpoint   = "unix://" + socket
	driverName = "com.ksc.csi.diskplugin"
	nodeID     = "test-node"
	version    = "0.1"
	// region           = "test-region"
	// availabilityZone = "test-availabilityzone"
)

type fakeNodeServer struct {
	*NodeServer
}

func getNodeServer(config *Config) *fakeNodeServer {
	nodeServer := &fakeNodeServer{
		NodeServer: &NodeServer{
			//driverName: config.DriverName,
			nodeName: nodeID,
			nodeID:   nodeID,
			mounter:  NewFakeMounter(),
		},
	}
	return nodeServer
}

type fakeControllerServer struct {
	*KscEBSControllerServer
}

func getControllerServer(config *Config) *fakeControllerServer {
	return &fakeControllerServer{
		KscEBSControllerServer: &KscEBSControllerServer{
			ebsClient: config.EbsClient,
			// kecClient:  config.KecClient,
			k8sClient: &fakeK8sClientWrap{},
		},
	}
}

// func (fc *fakeControllerServer) getNodeReginZone() (string, string, error) {
// 	return "test-region", "test-zone", nil
// }

type fakeIdentityServer struct {
	*IdentityServer
}

func getIdentityServer(config *Config) *fakeIdentityServer {
	return &fakeIdentityServer{
		IdentityServer: &IdentityServer{
			driverName: config.DriverName,
			version:    config.Version,
		},
	}
}

type fakeK8sClientWrap struct{}

func (fk *fakeK8sClientWrap) GetNodeRegionZone() (string, string, error) {
	return "test-region", "test-zone", nil
}
func (fk *fakeK8sClientWrap) IsNodeStatusReady(nodename string) (bool, error) {
	return false, nil
}
func getDriver(t *testing.T) *Driver {
	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove unix domain socket file %s, error: %s", socket, err)
	}

	driverConfig := &Config{
		EndPoint:   endpoint,
		DriverName: driverName,
		Version:    version,
		EbsClient:  NewFakeStorageClient(),
	}

	driver := &Driver{
		endpoint:         endpoint,
		identityServer:   getIdentityServer(driverConfig),
		controllerServer: getControllerServer(driverConfig),
		nodeServer:       getNodeServer(driverConfig),
		ready:            true,
	}
	return driver
}

func TestDriverSuite(t *testing.T) {
	d := getDriver(t)
	go d.Run()
	defer d.Stop()

	mntDir := os.TempDir()

	fmt.Println("mntDir:", mntDir)
	defer os.RemoveAll(mntDir)

	mntStageDir := os.TempDir()

	fmt.Println("mntStageDir:", mntStageDir)
	defer os.RemoveAll(mntStageDir)

	cfg := &sanity.Config{
		StagingPath: mntStageDir,
		TargetPath:  mntDir,
		Address:     endpoint,
	}

	sanity.Test(t, cfg)
}

type FakeStorageClient struct {
	volumes map[string]*ebsClient.Volume
}

func (cli *FakeStorageClient) DescribeInstanceVolumes(describeInstanceVolumesReq *ebsClient.DescribeInstanceVolumesReq) (*ebsClient.InstanceVolumes, error) {
	//TODO implement me
	panic("implement me")
}

func NewFakeStorageClient() *FakeStorageClient {
	volumes := make(map[string]*ebsClient.Volume)
	return &FakeStorageClient{
		volumes: volumes,
	}
}

// TODO
func (cli *FakeStorageClient) ExpandVolume(expandVolumeReq *ebsClient.ExpandVolumeReq) (*ebsClient.ExpandVolumeResp, error) {
	return nil, nil
	//listVolumesResp, err := cli.ListVolumes(expandVolumeReq)
	//if err != nil {
	//	return nil, err
	//}
	//if len(listVolumesResp.Volumes) == 0 {
	//	return nil, errors.New("not found volume")
	//}
	//return listVolumesResp.Volumes[0], nil
}

func (f *FakeStorageClient) ListVolumes(listVolumesReq *ebsClient.ListVolumesReq) (*ebsClient.ListVolumesResp, error) {
	volumes := make([]*ebsClient.Volume, 0)
	for _, volume := range f.volumes {
		volumes = append(volumes, volume)
	}
	listVolumesResp := &ebsClient.ListVolumesResp{
		RequestId: randString(32),
		Volumes:   volumes,
	}
	return listVolumesResp, nil
}

func (f *FakeStorageClient) GetVolume(listVolumesReq *ebsClient.ListVolumesReq) (*ebsClient.Volume, error) {
	vol, ok := f.volumes[listVolumesReq.VolumeIds[0]]
	if !ok {
		return nil, errors.New("volume not found")
	}

	return vol, nil
}
func (f *FakeStorageClient) GetVolumeByName(listVolumesReq *ebsClient.ListVolumesReq) (*ebsClient.ListVolumesResp, error) {

	return nil, nil
}

func (f *FakeStorageClient) CreateVolume(createVolumeReq *ebsClient.CreateVolumeReq) (*ebsClient.CreateVolumeResp, error) {
	id := randString(32)
	vol := &ebsClient.Volume{
		VolumeId:         id,
		AvailabilityZone: createVolumeReq.AvailabilityZone,
		VolumeName:       createVolumeReq.VolumeName,
		VolumeDesc:       createVolumeReq.VolumeDesc,
		Size:             createVolumeReq.Size,
		VolumeStatus:     ebsClient.AVAILABLE_STATUS,
	}
	f.volumes[id] = vol

	return &ebsClient.CreateVolumeResp{
		RequestId: randString(32),
		VolumeId:  vol.VolumeId,
	}, nil
}

func (f *FakeStorageClient) DeleteVolume(deleteVolumeReq *ebsClient.DeleteVolumeReq) (*ebsClient.DeleteVolumeResp, error) {
	delete(f.volumes, deleteVolumeReq.VolumeId)
	return &ebsClient.DeleteVolumeResp{}, nil
}

func (f *FakeStorageClient) Attach(attachVolumeReq *ebsClient.AttachVolumeReq) (*ebsClient.AttachVolumeResp, error) {
	vol, ok := f.volumes[attachVolumeReq.VolumeId]
	if !ok {
		return nil, fmt.Errorf("vol %v not found", attachVolumeReq.VolumeId)
	}
	vol.VolumeStatus = ebsClient.INUSE_STATUS
	f.volumes[vol.VolumeId] = vol

	return &ebsClient.AttachVolumeResp{
		RequestId: randString(32),
		Return:    true,
	}, nil
}

func (f *FakeStorageClient) Detach(detachVolumeReq *ebsClient.DetachVolumeReq) (*ebsClient.DetachVolumeResp, error) {
	vol, ok := f.volumes[detachVolumeReq.VolumeId]
	if !ok {
		return nil, fmt.Errorf("vol %v not found", detachVolumeReq.VolumeId)
	}
	vol.VolumeStatus = ebsClient.AVAILABLE_STATUS
	f.volumes[vol.VolumeId] = vol

	return &ebsClient.DetachVolumeResp{
		RequestId: "",
		Return:    true,
	}, nil
}

func (f *FakeStorageClient) ValidateAttachInstance(req *ebsClient.ValidateAttachInstanceReq) (*ebsClient.ValidateAttachInstanceResp, error) {
	return &ebsClient.ValidateAttachInstanceResp{
		RequestId:      randString(36),
		InstanceEnable: true,
	}, nil
}

func randString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

type fakeMounter struct{}

func NewFakeMounter() *fakeMounter {
	return &fakeMounter{}
}

func (f *fakeMounter) PathExists(path string) (bool, error) {
	return false, nil
}

func (f *fakeMounter) Expand(fsType, source string) (bool, error) {
	return false, nil
}
func (f *fakeMounter) Format(source string, fsType string) error {
	return nil
}

func (f *fakeMounter) Mount(source string, target string, fsType string, options ...string) error {
	return nil
}

func (f *fakeMounter) Unmount(target string) error {
	return nil
}

func (f *fakeMounter) IsFormatted(source string) (bool, error) {
	return true, nil
}
func (f *fakeMounter) IsMounted(target string) (bool, error) {
	return true, nil
}
