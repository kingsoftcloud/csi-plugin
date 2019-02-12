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

import (
	"encoding/json"
	"time"
)

type Cluster struct {
	ID             int32      `gorm:"AUTO_INCREMENT;primary_key" json:"-"`
	UUID           string     `gorm:"type:varchar(72);unique_index" json:"id"`
	Name           string     `gorm:"type:varchar(255)" json:"name"`
	Description    string     `gorm:"type:varchar(255)" json:"description"`
	Inner_slb_eip  string     `gorm:"type:varchar(32)" json:"inner_eip"`
	Inner_slb_port int        `json:"inner_port"`
	Outer_slb_eip  string     `gorm:"type:varchar(32)" json:"outer_eip"`
	Outer_slb_port int        `json:"outer_port"`
	Cidr           string     `gorm:"type:varchar(32)" json:"cidr" ql:"DEFAULT:'172.17.0.0/16'"`
	Service_Cidr   string     `gorm:"type:varchar(32)" json:"service_cidr" ql:"DEFAULT:'10.254.0.0/16'"`
	Vpc_id         string     `gorm:"type:varchar(72)" json:"vpc_id"`
	Vpc_name       string     `gorm:"type:varchar(72)" json:"vpc_name"`
	Cidr_block     string     `gorm:"type:varchar(72)" json:"cidr_block"`
	Password       string     `gorm:"type:varchar(255)" json:"password"`
	Tenant_id      string     `gorm:"type:varchar(32)" json:"tenant_id"`
	User_id        string     `gorm:"type:varchar(32)" json:"user_id"`
	Account_id     string     `gorm:"type:varchar(32)" json:"account_id"`
	Region         string     `json:"region"`
	Status         string     `gorm:"type:varchar(32)" json:"status" sql:"DEFAULT:'init'"`
	Version        string     `json:"version"`
	FeatureGates   string     `gorm:"type:varchar(1024)" json:"feature_gates"`
	Masters        []Nodes    `json:"masters"`
	Nodes          []Nodes    `json:"nodes"`
	Created_at     time.Time  `json:"created_at"`
	Updated_at     time.Time  `json:"updated_at"`
	Deleted_at     *time.Time `json:"deleted_at,omitempty"`
}

func (c *Cluster) GetFeatureGates() map[string]bool {
	featureGates := make(map[string]bool)
	if c.FeatureGates == "" {
		return featureGates
	}
	if err := json.Unmarshal([]byte(c.FeatureGates), &featureGates); err != nil {
		return featureGates
	}
	return featureGates
}
