package types

import "time"

type Certificate struct {
	ID           int32      `gorm:"AUTO_INCREMENT;primary_key" json:"-"`
	Cluster_uuid string     `gorm:"type:varchar(72)" json:"cluster_uuid"`
	Tenant_id    string     `gorm:"type:varchar(32)" json:"tenant_id"`
	Ca_key       string     `gorm:"type:text" json:"ca_key"`
	Ca_pem       string     `gorm:"type:text" json:"ca_pem"`
	Server_key   string     `gorm:"type:text" json:"server_key"`
	Server_pem   string     `gorm:"type:text" json:"server_pem"`
	Admin_key    string     `gorm:"type:text" json:"Admin_key"`
	Admin_pem    string     `gorm:"type:text" json:"Admin_pem"`
	Password     string     `gorm:"type:varchar(255)" json:"password"`
	Token        string     `gorm:"type:varchar(255)" json:"token"`
	Created_at   time.Time  `json:"created_at"`
	Updated_at   time.Time  `json:"updated_at"`
	Deleted_at   *time.Time `json:"deleted_at,omitempty"`
}
