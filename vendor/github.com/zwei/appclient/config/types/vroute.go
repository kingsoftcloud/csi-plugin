package types

import "time"

type Vroute struct {
	ID            int32      `gorm:"AUTO_INCREMENT;primary_key" json:"-"`
	Cluster_uuid  string     `gorm:"type:varchar(72)" json:"cluster_uuid"`
	Instance_uuid string     `gorm:"type:varchar(72)" json:"instance_uuid"`
	Vroute_id     string     `gorm:"type:varchar(255)" json:"vroute_id"`
	Subnet        string     `gorm:"type:varchar(32)" json:"subnet"`
	Created_at    time.Time  `json:"created_at"`
	Updated_at    time.Time  `json:"updated_at"`
	Deleted_at    *time.Time `json:"deleted_at,omitempty"`
}
