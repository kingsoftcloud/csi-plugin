# appclient

> appclient 是 appengine 的客户端

## appclient 使用手册

### 接口使用方式

```bash
# 使用默认的endpoint 地址（appclient 服务地址)
import (
	"fmt"
	"github.com/golang/glog"
	"github.com/zwei/appclient"
)

func main() {
	conf := appclient.NewDefaultConfig()

	// get appclient version info
	version, err := appclient.Version(conf)
	if err != nil {
		glog.Errorf("init appclient error %s", err)
	}

	appclientVersion, err := version.GetVersion()
	if err != nil {
		glog.Errorf("connect appclient err %s", err)
	}

	// 打印 appengine 版本号
	fmt.Sprintln(appclientVersion)

	// 获取本机的信息和所在的集群id
	node, err := appclient.Node(conf)
	if err != nil {
		glog.Errorf("init appclient error %s", err)
	}

	//获取本机的信息
	nodeInfo, err := node.GetLocalNode()
	if err != nil {
		glog.Errorf("connect appclient err %s", err)
	}

	// 打印node 信息 instance uuid， fixip
	fmt.Sprintln(nodeInfo.Instance_uuid, nodeInfo.Instance_fixip)

	//获取本机所在集群的信息
	cluster, err := appclient.Cluster(conf)
	if err != nil {
		glog.Errorf("init appclient error %s", err)
	}

	clusterInfo, err := cluster.GetCluster(nodeInfo.Cluster_uuid)
	if err != nil {
		glog.Errorf("connect appclient err %s", err)
	}

	//打印集群信息
	fmt.Sprintln(clusterInfo)
}
```