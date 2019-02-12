/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package types

import "time"

type Plugins struct {
	// k8s plugins info
	Plugins []Plugin `json:"plugins"`
}

type PluginOperate struct {
	// yaml image update
	Update bool `json:"update"`
	// yaml delete
	Delete bool `json:"delete"`
	// yaml delete and yaml create
	Reset bool `json:"reset"`
}

type Plugin struct {
	ID          int32      `gorm:"AUTO_INCREMENT;primary_key" json:"-"`
	UUID        string     `gorm:"type:varchar(72) ;unique_index" json:"uuid"`
	Name        string     `gorm:"type:varchar(255)" json:"name"`
	Version     string     `gorm:"type:varchar(255)" json:"version"`
	Status      string     `gorm:"type:varchar(16)" json:"status"`
	Task_Status string     `gorm:"type:varchar(16)" json:"task_status" sql:"DEFAULT:'ready'"`
	Location    string     `gorm:"type:varchar(255)" json:"location"`
	Operate     string     `gorm:"type:varchar(255)" json:"operate" sql:"DEFAULT:'{}'"`
	Created_at  time.Time  `json:"created_at"`
	Updated_at  time.Time  `json:"updated_at"`
	Deleted_at  *time.Time `json:"deleted_at,omitempty"`
}

type PluginMetaData struct {
	ID           int32      `gorm:"AUTO_INCREMENT;primary_key" json:"-"`
	Name         string     `gorm:"type:varchar(255)" json:"name"`
	PluginUUID   string     `gorm:"type:varchar(72)" json:"plugin_uuid"`
	Cluster_uuid string     `gorm:"type:varchar(72)" json:"cluster_uuid"`
	Status       string     `gorm:"type:varchar(16)" json:"status"`
	Task_Status  string     `gorm:"type:varchar(16)" json:"task_status" sql:"DEFAULT:'ready'"`
	Created_at   time.Time  `json:"created_at"`
	Updated_at   time.Time  `json:"updated_at"`
	Deleted_at   *time.Time `json:"deleted_at,omitempty"`
}
