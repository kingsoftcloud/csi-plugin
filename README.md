# ksc csi diskplugin

金山云CSI插件， 实现了在Kubernetes集群中对金山云云存储卷的生命周期管理，支持动态创建、挂载、使用云数据卷。

**注意：** 
1. 由于金山云硬盘不支持跨可用区挂载，所以要保证集群的所有node节点都在同一个可用区。
2. 支持的k8s版本: v1.17 以上

## 服务编译和发布

1. 构建镜像： 

修改Makefile，将“yourcipherkey”改为实际的密钥，若SK无需加密，则设置为空

例如KEY='404633a025a386e110d54242a48f885e'，则
```
CIPHER_KEY=$(shell echo "404633a025a386e110d54242a48f885e")
```
若不加密，则
```
CIPHER_KEY=$(shell echo "")
```
修改完成后执行
```sh
make build
```    
2. 推送镜像：

请将Makefile中镜像仓库地址修改为自己实际的地址
```
BJKSYUNREPOSITORY:= hub.kce.ksyun.com/ksyun
```
修改完成后执行
```
make push
```

## 配置AKSK
示例：
```yaml
AK："AKTRQxqRY0SdCw31S46rrcMA"
SK："ODPedeQvrIo2BF6QkzkZ1HZdhkjH648cOF0fVXGt"
KEY: "404633a025a386e110d54242a48f885e"（32位）
```
### 1. 如果需要对SK加密，则执行以下操作：
注意：如果不需要对SK加密，请忽略此步

* 首先将KEY字符串转换16进制：

string转16进制命令：
```sh
# echo -n '404633a025a386e110d54242a48f885e' | xxd -p
3430343633336130323561333836653131306435343234326134386638383565
```
* 执行加密命令：
```sh
#echo -n "ODPedeQvrIo2BF6QkzkZ1HZdhkjH648cOF0fVXGt" |openssl enc -aes-256-cbc -e -a -K 3430343633336130323561333836653131306435343234326134386638383565 -iv 34303436333361303235613338366531
```
参数说明：

-e 加密

-a  加密后以base64编码

-K 加密key （16进制）

-iv iv值(固定长度：16位)   （16进制）取密钥key的前16位作为iv值

加密后字符串：
```sh
70aM3hAdVJMB/yJHOxIB3iHyST0aijaIQWoIXCo6yLgFRofS2lHs62Q0Z6wAhgY+
```

### 2. 创建secret，将AK、SK保存在其中
* 当对SK加密时：
```sh
# kubectl create secret generic kce-security-token --from-literal=ak='AKTRQxqRY0SdCw31S46rrcMA' --from-literal=sk='70aM3hAdVJMB/yJHOxIB3iHyST0aijaIQWoIXCo6yLgFRofS2lHs62Q0Z6wAhgY+' --from-literal=cipher='aes256+base64' -n kube-system
```
* 当不对SK加密时：
```sh
# kubectl create secret generic kce-security-token --from-literal=ak='AKTRQxqRY0SdCw31S46rrcMA' --from-literal=sk='ODPedeQvrIo2BF6QkzkZ1HZdhkjH648cOF0fVXGt' -n kube-system
```

## 部署：
1. 配置kubectl命令连接到集群
2. 修改deploy目录下controller-plugin.yaml和node-plugin.yaml文件中csi-diskplugin容器的镜像地址为您的实际地址
```
image: hub.kce.ksyun.com/ksyun/csi-diskplugin:1.9.1-open
imagePullPolicy: Always
name: csi-diskplugin
```
3. 修改[aksk-configmap.yaml](deploy/aksk-configmap.yaml)中字段为您的实际情况
```yaml
data:
  aksk-conf: |
    {
      "region": "cn-beijing-6",
      "aksk_type": "file",
      "aksk_file_path": "",
    }

```
4. 执行 `make deploy_all` 部署csi插件服务

## 通过 helm chart 部署
1. 修改 values.yaml 中镜像地址
```
image:
   registry: hub.kce.ksyun.com
   pullPolicy: IfNotPresent
   namespace: kube-system
   driver: com.ksc.csi.diskplugin,com.ksc.csi.nfsplugin,com.ksc.csi.ks3plugin
   controller:
   repo: ksyun/csi-diskplugin
   tag: 1.8.15-mp
```
2. 修改 values.yaml kubeletDir、region 与 aksk 信息为您的实际情况
```yaml
kubeletDir: /data/kubelet
region: cn-beijing-6

aksk:
  source: configMap  # 或 secret
  name: user-temp-aksk
  mountPath: /var/lib/aksk
```

## 使用 csi plugin 存储卷

#### 动态存储卷

**创建 StorageClass**

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: test-csi-provisioner
provisioner: ksc/ebs
parameters:
  type: SSD3.0
  region: cn-beijing-6
  zone: cn-beijing-6a
  chargetype: Daily
  purchasetime: "10"
```
> 参数说明： 
> - type: EBS类型，必填，可选参数：SSD2.0/SSD3.0/SATA2.0/SATA3.0（字母全部大写）。
> - region: 创建云盘的集群地域
> - zone: 创建云盘的可用区，注意不同可用区可创建的EBS类型不一样，具体对应关系参考 [云硬盘使用限制](https://docs.ksyun.com/documents/5423)，默认值是插件所在node的可用区。
> - chargetype: 云盘的计费方式，默认值为Daliy，详情参考[创建云硬盘Open Api](https://docs.ksyun.com/documents/5446)中的chargetype字段。
> - purchasetime: 若选择"包年包月"的计费方式，需要设置购买时长，单位为月, 默认值是 1 月。
> - projectid: 创建云盘所在的项目ID，默认值是默认项目。


**创建 pvc**
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: nginx-pvc
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: test-csi-provisioner 
  resources:
    requests:
      storage: 20Gi
```
> accessModes：只支持 ReadWriteOnce

**创建 pod，使用pvc**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: web-server
spec:
  containers:
   - name: web-server
     image: nginx 
     volumeMounts:
       - mountPath: /usr/share/nginx/html
         name: mypvc
  volumes:
   - name: mypvc
     persistentVolumeClaim:
       claimName: nginx-pvc
       readOnly: false
```
