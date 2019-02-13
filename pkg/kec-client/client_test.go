package kecClient

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

func getKecClient() *Client {
	OpenApiConfig := &api.ClientConfig{
		AccessKeyId:     "AKLTd3j9wnDnSamjGtU4Ngj8og",
		AccessKeySecret: "ON9XNwu+DFCOhbmABbCQmVm9eldy8EkeOKw0lIKH462fkDPb5jBvUGw67vW5aaSHhw==",
		OpenApiEndpoint: "api.ksyun.com",
		OpenApiPrefix:   "https",
		Region:          "cn-beijing-6",
	}
	return New(OpenApiConfig)
}

func TestKecClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ebsClient Suite")
}

var _ = Describe("Test KecClient", func() {
	var client *Client
	BeforeEach(func() {
		client = getKecClient()
	})

	Describe("get kec instance info", func() {
		var instanceUUID = ""
		It("should fail when no instance uuid is provided", func() {
			kecInfo, err := client.DescribeInstances(instanceUUID)
			glog.Info(err)
			Expect(err).To(HaveOccurred())
			Expect(kecInfo).To(BeNil())
		})
	})
})
