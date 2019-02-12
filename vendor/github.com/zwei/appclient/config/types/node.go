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

type Node struct {
	ID             int32      `gorm:"AUTO_INCREMENT;primary_key" json:"-"`
	Cluster_uuid   string     `gorm:"type:varchar(72)" json:"cluster_uuid"`
	Instance_uuid  string     `gorm:"type:varchar(72)" json:"instance_uuid"`
	Instance_fixip string     `gorm:"type:varchar(16)" json:"instance_fixip"`
	Type           string     `gorm:"type:varchar(16)" json:"type"`
	Status         string     `gorm:"type:varchar(16)" json:"status" sql:"DEFAULT:'init'"`
	Task_Status    string     `gorm:"type:varchar(16)" json:"task_status" sql:"DEFAULT:'ready'"`
	Error_stage    string     `gorm:"type:varchar(16)" json:"error_stage" sql:"DEFAULT:''"`
	Error_msg      string     `gorm:"type:varchar(255)" json:"error_msg" sql:"DEFAULT:''"`
	Created_at     time.Time  `json:"created_at"`
	Updated_at     time.Time  `json:"updated_at"`
	Deleted_at     *time.Time `json:"deleted_at,omitempty"`
}

type Nodes struct {
	// nova instance uuid
	Instance_uuid string `json:"instance_uuid"`
	// node instance fixip
	Instance_fixip string `json:"ip"`
}
