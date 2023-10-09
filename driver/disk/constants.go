package driver

const (
	// volume type
	SSD2_0  string = "SSD2.0"
	SSD3_0  string = "SSD3.0"
	SATA3_0 string = "SATA3.0"
	EHDD    string = "EHDD"
	ESSD    string = "ESSD"

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
)
