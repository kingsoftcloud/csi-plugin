package types

import "time"

type EtcdLeader struct {
	ID           int32      `gorm:"AUTO_INCREMENT;primary_key" json:"-"`
	Cluster_uuid string     `gorm:"type:varchar(72)" json:"cluster_uuid"`
	EtcdLeader   string     `gorm:"type:varchar(72)" json:"etcdleader"`
	MetaData     string     `gorm:"type:varchar(255)" json:"meta_data"`
	Created_at   time.Time  `json:"created_at"`
	Updated_at   time.Time  `json:"updated_at"`
	Deleted_at   *time.Time `json:"deleted_at,omitempty"`
}

type EtcdLocation struct {
	ID           int32      `gorm:"AUTO_INCREMENT;primary_key" json:"-"`
	Cluster_uuid string     `gorm:"type:varchar(72)" json:"cluster_uuid"`
	Snap_uuid    string     `gorm:"type:varchar(72)" json:"snap_uuid"`
	Location     string     `gorm:"type:varchar(255)" json:"location"`
	Size         int64      `json:"size"`
	Checksum     string     `gorm:"type:varchar(255)" json:"checksum"`
	Status       string     `gorm:"type:varchar(255)" json:"status"`
	Created_at   time.Time  `json:"created_at"`
	Updated_at   time.Time  `json:"updated_at"`
	Deleted_at   *time.Time `json:"deleted_at,omitempty"`
}

type EtcdLocations struct {
	// one cluster all etcd snapshot
	Locations []EtcdLocation `json:"locations"`
}
