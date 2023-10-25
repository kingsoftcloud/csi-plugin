package driver

import "time"

const (
	// volume type
	SSD2_0   string = "SSD2.0"
	SSD3_0   string = "SSD3.0"
	SATA3_0  string = "SATA3.0"
	EHDD     string = "EHDD"
	ESSD     string = "ESSD"
	ESSD_PL1 string = "ESSD_PL1"
	ESSD_PL2 string = "ESSD_PL2"
	ESSD_PL3 string = "ESSD_PL3"
	ESSD_PL0 string = "ESSD_PL0"

	// ESSD_PERFORMANCE_LEVEL is storage class
	ESSD_PERFORMANCE_LEVEL = "performanceLevel"

	DISK_PERFORMANCE_LEVEL0 = "PL0"
	DISK_PERFORMANCE_LEVEL1 = "PL1"
	DISK_PERFORMANCE_LEVEL2 = "PL2"
	DISK_PERFORMANCE_LEVEL3 = "PL3"

	// NodeSchedueTag in annotations
	NodeSchedueTag = "volume.kubernetes.io/selected-node"

	nodeStorageLabel = "com.ksc.csi.node/disktype.%s"

	labelVolumeType   = "com.ksc.csi.node/disktype"
	annAppendPrefix   = "com.ksc.csi.node/annotation-prefix/"
	annVolumeTopoKey  = "com.ksc.csi.node/volume-topology"
	labelAppendPrefix = "com.ksc.csi.node/label-prefix/"

	UpdateNodeTimeout  = 1 * time.Hour
	GetDiskTypeTimeout = 30 * time.Minute

	KubeNodeName = "KUBE_NODE_NAME"

	// instanceTypeLabel ...
	instanceTypeLabel = "beta.kubernetes.io/instance-type"

	NodeRegionKey = "failure-domain.beta.kubernetes.io/region"
	NodeZoneKey   = "failure-domain.beta.kubernetes.io/zone"

	// KceLabel instance type ...
	KceInstanceTypeLabel = "kce/machine-model"
	// KceLabel zone ....
	KceLabelZoneKey = "kce/kec-zone"

	InstanceUuid           = "appengine.sdns.ksyun.com/instance-uuid"
	NodeAnnotationNodeType = "appengine.sdns.ksyun.com/node-type"
)
