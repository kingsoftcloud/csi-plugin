# ksc csi diskplugin

实现了金山云 EBS 云硬盘和集群 csi 接口的对接，完成在k8s集群通过 storageclass 和 pvc 资源自动的从金山云硬盘创建并挂载对应的pv持久化存储卷。

**注意：** 
1. 由于金山云硬盘不支持跨可用区挂载，所以要保证集群的所有node节点都在同一个可用区。
2. 支持的k8s版本: v1.12


### 服务编译和发布

构建镜像：

    make build VERSION=v0.1.0
    
推送镜像：

    make push VERSION=v0.1.0

单元测试：

    make test

部署：
1. 配置kubectl命令连接到集群
2. 执行 `make deploy_v0.1.0` 部署csi插件服务


### 使用 csi plugin 存储卷

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
