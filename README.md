# ksc csi diskplugin

实现了金山云 EBS 云硬盘和集群 csi 接口的对接，完成在k8s集群通过 storageclass 和 pvc 资源自动的从金山云硬盘创建并挂载对应的pv持久化存储卷。

**注意：** 由于金山云硬盘不支持跨可用区挂载，所以要保证集群的所有node节点都在同一个可用区。


构建镜像：

    make build VERSION=v0.1.0
    
推送镜像：

    make push VERSION=v0.1.0

单元测试：

    make test

部署：
1. 配置kubectl命令连接到集群
2. 执行 `make deploy_v0.1.0` 部署csi插件服务

