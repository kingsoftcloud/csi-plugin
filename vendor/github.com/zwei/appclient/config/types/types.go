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

// plugins status
const (
	PLUGINS_READY         = "ready"
	PLUGINS_ERROR         = "error"
	PLUGINS_DELETED       = "deleted"
	PLUGINS_UPDATED       = "updated"
	PLUGINS_TASK_DELETING = "deleting"
	PLUGINS_TASK_UPDATING = "updating"
	PLUGINS_TASK_READY    = "ready"
)

// cluster status
const (
	CLUSTERINIT     = "init"
	CLUSTERREADY    = "ready"
	CLUSTERACTIVE   = "ready"
	CLUSTERBUILD    = "building"
	CLUSTERERROR    = "error"
	CLUSTERDELETING = "deleting"
	CLUSTERDELETED  = "deleted"
)

// node status
const (
	NODE_INIT     = "init"
	NODE_ERROR    = "error"
	NODE_READY    = "ready"
	NODE_NOTREADY = "notready"
	NODE_DELETED  = "deleted"
	NODE_UPDATED  = "updated"

	NODE_TASK_READY   = "ready"
	NODE_TASK_ERROR   = "error"
	NODE_TASK_CHECK   = "checking"
	NODE_TASK_INSTALL = "installing"
	NODE_TASK_DELETE  = "deleting"
	NODE_TASK_RESET   = "resetting"
	NODE_TASK_UPDATE  = "updating"
)

// etcd status
const (
	ETCDSNAP_INIT    = "init"
	ETCDSNAP_READY   = "ready"
	ETCDSNAP_SAVE    = "saving"
	ETCDSNAP_ERROR   = "error"
	ETCDSNAP_DELETED = "deleted"
)

type Info struct {
	Version   string `json:"Version"`
	GoVersion string `json:"goVersion"`
	Compiler  string `json:"compiler"`
	Platform  string `json:"platform"`
}

const (
	DefaultKubeApiServer         = "kube-apiserver"
	DefaultKubeControllerManager = "kube-controller-manager"
	DefaultKubeScheduler         = "kube-scheduler"
	DefaultKubeProxy             = "kube-proxy"
	DefaultKubelet               = "kubelet"
	DefualtDocker                = "docker"
	DefaultKubeEtcd              = "etcd"
	DefaultCanalFlanneld         = "canal-flanneld"
	DefaultKubeDNS               = "kube-dns"

	DefaultPlugins = "plugins"
	DefaultDemon   = "demon"
	DefaultHosts   = "hosts"
)

// node error message during installation
type ErrorInfo struct {
	ErrorStage string `json:"error_stage"`
	ErrorMsg   string `json:"error_msg"`
}

const (
	CHECK_SYSTEM_ADDIPTABLES   = "check_system_addiptables"
	CHECK_HOST_SYNCHOSTS       = "check_host_synchosts"
	CHECK_DOCKER_DOCKERCHECK   = "check_docker_dockercheck"
	CHECK_KUBECTL_KUBECTLCHECK = "check_kubectl_kubectlcheck"
	CHECK_KUBELET_KUBELETCHECK = "check_kubelet_kubeletcheck"
	CHECK_LABEL_MKLABEL        = "check_label_mklabel"
	CHECK_YAML_YAML_CHECK      = "check_yaml_yamlcheck"
	CHECK_PLUGINS_INITPLUGINS  = "check_plugins_initplugins"
	CHECK_ETCDSNAP_RUNETCDSNAP = "check_etcdsnap_runetcdsnap"

	DELETE_LABEL_DELETENODE         = "delete_label_deletenode"
	DELETE_SYSTEM_DELETESYSTEM      = "delete_system_deletesystem"
	DELETE_HOSTS_DELETEHOSTS        = "delete_hosts_deletehosts"
	DELETE_KUBELET_KUBELETUNINSTALL = "delete_kubelet_kubeletuninstall"
	DELETE_DOCKER_DOCKERUNINSTALL   = "delete_docker_dockeruninstall"
	DELETE_KUBECTL_KUBECTLUNINSTALL = "delete_kubectl_kubectluninstall"

	INSTALL_SYSTEM_CHECKSYSTEM     = "install_system_checksystem"
	INSTALL_HOSTS_SYNCHOSTS        = "install_hosts_synchosts"
	INSTALL_DOCKER_DOCKERINSTALL   = "install_docker_dockerinstall"
	INSTALL_KUBECTL_KUBECTLINSTALL = "install_kubectl_kubectlinstall"
	INSTALL_KUBELET_KUBELETINSTALL = "install_kubelet_kubeletinstall"
	INSTALL_YAML_YAMLINSTALL       = "install_yaml_yamlinstall"
	INSTALL_LABEL_MKLABEL          = "install_label_mklabel"

	RESET_RESET   = "reset_reset"
	UPDATE_UPDATE = "update_update"
)
