package types

import "time"

type CrtLocation struct {
	ID           int32      `gorm:"AUTO_INCREMENT;primary_key" json:"-"`
	Cluster_uuid string     `gorm:"type:varchar(72)" json:"cluster_uuid"`
	Slb_eip      string     `gorm:"type:varchar(32)" json:"slb_eip"`
	Slb_type     string     `gorm:"type:varchar(32)" json:"slb_type"`
	Value        string     `gorm:"type:varchar(255)" json:"value"`
	Created_at   time.Time  `json:"created_at"`
	Updated_at   time.Time  `json:"updated_at"`
	Deleted_at   *time.Time `json:"deleted_at,omitempty"`
}
