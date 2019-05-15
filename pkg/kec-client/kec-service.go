package kecClient

type KecService interface {
	DescribeInstances(instance_id string) (*KecInfo, error)
}

type KecInfo struct {
	InstanceId       string `json:"InstanceId"`
	AvailabilityZone string `json:"AvailabilityZone"`
}

type KecList struct {
	Instances []KecInfo `json:"InstancesSet"`
}
