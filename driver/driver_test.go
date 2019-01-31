package driver

import (
	"flag"
	"fmt"
	"io/ioutil"
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
	region     = "test-region"
)

func getDriver(t *testing.T) *Driver {
	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove unix domain socket file %s, error: %s", socket, err)
	}

	driverConfig := &DriverConfig{
		EndPoint:   endpoint,
		DriverName: driverName,
		NodeID:     nodeID,
		Version:    version,
		Region:     region,
	}

	return NewDriver(driverConfig, NewFakeStorageClient(), nil)
}
func TestDriverSuite(t *testing.T) {
	d := getDriver(t)
	go d.Run()
	defer d.Stop()

	mntDir, err := ioutil.TempDir("", "mnt")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("mntDir:", mntDir)
	defer os.RemoveAll(mntDir)

	mntStageDir, err := ioutil.TempDir("", "mnt-stage")
	if err != nil {
		t.Fatal(err)
	}
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

func NewFakeStorageClient() *FakeStorageClient {
	volumes := make(map[string]*ebsClient.Volume)
	return &FakeStorageClient{
		volumes: volumes,
	}
}

func (f *FakeStorageClient) ListVolumes(listVolumesReq *ebsClient.ListVolumesReq) (*ebsClient.ListVolumesResp, error) {
	volumes := make([]*ebsClient.Volume, 0)
	for _, volume := range f.volumes {
		volumes = append(volumes, volume)
	}
	listVolumesResp := &ebsClient.ListVolumesResp{
		RequestId: randString(10),
		Volumes:   volumes,
	}
	return listVolumesResp, nil
}

func (f *FakeStorageClient) GetVolume(listVolumesReq *ebsClient.ListVolumesReq) (*ebsClient.Volume, error) {
	return nil, nil
}

func (f *FakeStorageClient) CreateVolume(createVolumeReq *ebsClient.CreateVolumeReq) (*ebsClient.CreateVolumeResp, error) {
	if err := ebsClient.ValidateCreateVolumeReq(createVolumeReq); err != nil {
		return nil, err
	}

	id := randString(10)
	vol := &ebsClient.Volume{
		VolumeId:         id,
		AvailabilityZone: createVolumeReq.AvailabilityZone,
		VolumeName:       createVolumeReq.VolumeName,
		VolumeDesc:       createVolumeReq.VolumeDesc,
		Size:             createVolumeReq.Size,
	}
	f.volumes[id] = vol

	return &ebsClient.CreateVolumeResp{
		RequestId: randString(10),
		VolumeId:  vol.VolumeId,
	}, nil
}

func (f *FakeStorageClient) DeleteVolume(deleteVolumeReq *ebsClient.DeleteVolumeReq) (*ebsClient.DeleteVolumeResp, error) {
	delete(f.volumes, deleteVolumeReq.VolumeId)
	return &ebsClient.DeleteVolumeResp{}, nil
}

func (f *FakeStorageClient) Attach(attachVolumeReq *ebsClient.AttachVolumeReq) (*ebsClient.AttachVolumeResp, error) {
	return nil, nil
}

func (f *FakeStorageClient) Detach(detachVolumeReq *ebsClient.DetachVolumeReq) (*ebsClient.DetachVolumeResp, error) {
	return nil, nil
}

func randString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
