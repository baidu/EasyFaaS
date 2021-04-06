# 使用docker-compose在本地运行openless服务 

以下教程一步一步教你如何使用docker-compose在本地快速运行openless服务


## 1. 准备工作 
### 1.1基础环境 
**1.1.1 操作系统**： linux： 内核4.0+

**1.1.2 容器环境**：

​		确保本地Docker服务正常运行，因openless服务底层依赖cgroup/loop设备等linux底层特性建议优先使用Linux系统。

​		如果您想在MacOS系统运行，请使用docker-machine安装Docker环境，避免使用docker for Mac等直接安装。

*环境安装参考文档：*

- [安装docker](https://docs.docker.com/engine/install/)
- [安装docker-compose](https://docs.docker.com/compose/install/)

### 1.3 openless组件镜像
如总体架构图所述，openless由4个组件构成，分别是
controller组件， funclet组件， runner-runtime组件，以及stubs模块构建的本地代码仓库组件func-registry。

您可以直接使用openless发布的公共镜像，如果您本机无法访问公网, 可在能访问外网环境的服务器上下载并存储镜像文件到tar文件中，再将tar文件load至无公网的服务器。

*您也可以先跳过此步骤，直接用我们提供的镜像案例运行。*


## 2. 运行openless服务
### 2.1 设置openless服务组件容器运行参数
主要配置在./scripts/docker-compose/compose-local/.env 文件，其中 
-  **CONTROLLER_EXPORT_PORT**: controller组件暴露的服务端口
-  **REGISTRY_EXPORT_PORT**： mock本地函数代码仓库服务组件的服务端口
-  **FUNCTION_DATA_DIR**: 本地函数代码目录
-  **OPENLESS_REPO**： 镜像仓库repo
-  **CONTROLLER_TAG**： controller组件镜像tag, 完整镜像ID为${OPENLESS_REPO}/controller:{$CONTROLLER_TAG}
-  **FUNCLET_TAG**： funclet组件镜像tag, 完整镜像ID为${OPENLESS_REPO}/mini-funclet:{$CONTROLLER_TAG}
-  **RUNTIME_TAG**: runner-runtime组件镜像tag, 完整镜像ID为${OPENLESS_REPO}/runner-runtime:{$CONTROLLER_TAG}
-  **REGISTRY_TAG**: 本地mock函数仓库组件tag, 完整镜像ID为${OPENLESS_REPO}//func-registry:{$CONTROLLER_TAG}

```cassandraql
# env文件配置 示例
$ cat ./scripts/docker-compose/compose-local/.env 
CONTROLLER_EXPORT_PORT=8899
REGISTRY_EXPORT_PORT=8002
FUNCTION_DATA_DIR=/tmp/funcData
OPENLESS_REPO=registry.baidubce.com/openless-public
CONTROLLER_TAG=demo1.0
FUNCLET_TAG=demo1.0
RUNTIME_TAG=demo1.0
REGISTRY_TAG=demo1.0
```
备注：您也可以跳过此步骤，直接运行脚本快速体验（确保端口号不能冲突）

### 2.2 启动服务
运行如下命令启动服务，并运行docker ps查看组件是否正确运行：
其中controller组件容器，funclet组件容器及func-registry组件容器为常驻容器，
查看容器是否正常运行
```
# 启动服务 
$ cd ./scripts/docker-compose/compose-local/
$ sh app_control.sh start 

# 检查服务
$ docker ps 
CONTAINER ID        IMAGE                                       COMMAND                  CREATED             STATUS              PORTS               NAMES
5bec1b3fd3e4        controller:demo1.0        "./controller --maxproc…" 3 days ago          Up 3 days                               minikun_minikun_1
8b6b1f6d2b1d        mini-funclet:demo1.0      "./funclet --logtost…"    3 days ago          Up 3 days                               minikun_funclet_1
5082b7249d28        func-registry:demo1.0     "/stubs --port=8002 …"    3 days ago          Up 3 days                               minikun_registry_1

 $ 查看服务是否正常运行 
```

## 3. 查看运行
详细见 [操作教程](./op_func.md) 




