package ebsClient

import (
	api "csi-plugin/pkg/open-api"
	"flag"
	"math/rand"
	"testing"
	"time"

	"github.com/golang/glog"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	flag.Set("logtostderr", "true")
	flag.Set("v", "5")
}

func getEbsClient() *Client {
	OpenApiConfig := &api.ClientConfig{
		AccessKeyId:     "AKLTd3j9wnDnSamjGtU4Ngj8og",
		AccessKeySecret: "ON9XNwu+DFCOhbmABbCQmVm9eldy8EkeOKw0lIKH462fkDPb5jBvUGw67vW5aaSHhw==",
		OpenApiEndpoint: "api.ksyun.com",
		OpenApiPrefix:   "https",
		Region:          "cn-beijing-6",
	}
	return New(OpenApiConfig)
}

func TestEbsClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ebsClient Suite")
}

var _ = Describe("EbcClient", func() {
	var client *Client
	BeforeEach(func() {
		client = getEbsClient()
	})
	Describe("List volumes", func() {
		It("success list volumes", func() {
			req := &ListVolumesReq{}
			resp, err := client.ListVolumes(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Volumes).NotTo(BeNil())
		})
	})

	Describe("create volume", func() {
		It("should fail when no VolumeType is provided", func() {
			req := &CreateVolumeReq{
				Size:             10,
				AvailabilityZone: "cn-beijing-6a",
				ChargeType:       "HourlyInstantSettlement",
			}
			resp, err := client.CreateVolume(req)
			glog.Info("err", err)
			Expect(err).To(HaveOccurred())
			Expect(resp).To(BeNil())
		})
		It("should fail when invalid VolumeType is provided", func() {
			req := &CreateVolumeReq{
				VolumeType:       "test-ssd",
				Size:             10,
				AvailabilityZone: "cn-beijing-6a",
				ChargeType:       "HourlyInstantSettlement",
			}
			resp, err := client.CreateVolume(req)
			glog.Info("err", err)
			Expect(err).To(HaveOccurred())
			Expect(resp).To(BeNil())
		})
		It("should fail when no AvailabilityZone is provided", func() {
			req := &CreateVolumeReq{
				Size:       10,
				VolumeType: "SSD3.0",
				ChargeType: "HourlyInstantSettlement",
			}
			resp, err := client.CreateVolume(req)
			glog.Info("err", err)
			Expect(err).To(HaveOccurred())
			Expect(resp).To(BeNil())
		})
		It("should fail when no ChargeType is provided", func() {
			req := &CreateVolumeReq{
				Size:             10,
				VolumeType:       "SSD3.0",
				AvailabilityZone: "cn-beijing-6a",
			}
			resp, err := client.CreateVolume(req)
			glog.Info("err", err)
			Expect(err).To(HaveOccurred())
			Expect(resp).To(BeNil())
		})

		It("should fail when invalid PurchaseTime is provided", func() {
			req := &CreateVolumeReq{
				VolumeName:       "test-volume",
				VolumeType:       "SSD3.0",
				Size:             10,
				AvailabilityZone: "cn-beijing-6a",
				ChargeType:       "Daily",
				PurchaseTime:     39,
			}
			resp, err := client.CreateVolume(req)

			glog.Info("err", err)
			Expect(err).To(HaveOccurred())
			Expect(resp).To(BeNil())
		})

		It("should success create volume", func() {
			req := &CreateVolumeReq{
				VolumeName:       "test-volume1",
				VolumeType:       "SSD3.0",
				Size:             10,
				AvailabilityZone: "cn-beijing-6a",
				ChargeType:       "HourlyInstantSettlement",
			}
			resp, err := client.CreateVolume(req)

			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
			Expect(resp.VolumeId).NotTo(BeEmpty())

			WaitVolumeStatus(client, resp.VolumeId, AVAILABLE_STATUS)

			By("cleaning up deleting the volume")
			deleteVolumeReq := &DeleteVolumeReq{
				resp.VolumeId,
			}
			deleteVolumeResp, err := client.DeleteVolume(deleteVolumeReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(deleteVolumeResp).NotTo(BeNil())
			Expect(deleteVolumeResp.Return).To(BeTrue())
		})
	})
})
