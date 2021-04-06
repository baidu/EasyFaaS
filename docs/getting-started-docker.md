## 1. 编译

1.1 编译全部模块
备注：如果您是在linux环境或者不需要对单个模块进行编译，只需要运行1.1的命令即可

```$shell
$ make
```

1.2 为其它平台交叉编译：

```bash
$ KUN_BUILD_PLATFORMS="linux/amd64" make
```

1.3 只编译某一个模块：

```bash
$ make WHAT=./pkg/controller
```


## 2. 打包

用法

```$shell
$ scripts/build-image.sh -h
Usage: scripts/build-image.sh [OPTIONS]
  -h  --help      帮助
  -m= --module=   模块名称
  -i= --image=    镜像名称
  -t= --tag=      镜像标签
  -e= --env=      部署环境（镜像仓库配置）
```

您需要依次打包controller组件、 funclet组件、 stubs组件，需要注意的runtime目前不开源，直接拉取镜像即可
执行脚本打包

```$shell
$ scripts/build-image.sh -m=controller -i=mini-controller -t=dev
$ scripts/build-image.sh -m=funclet -i=mini-funclet -t=dev
$ scripts/build-image.sh -m=stubs -i=func-registry -t=dev
```

如果您需要发布打包好的镜像，需要指定REGISTRY，如：

```$shell
$REGISTRY=registry.baidubce.com/<your_namespace>/ ./scripts/build-image.sh -m=funclet -i=mini-funclet -t=dev
```

REGISTRY地址可以在 [百度云容器镜像服务](https://cloud.baidu.com/doc/CCR/s/qk8gwqs4a) 中免费申请体验


如果需要容器build项目，可以使用WITHBUILD环境变量控制

```$shell
$ WITHBUILD=1 scripts/build-image.sh -e=test -m=funclet -i=mini-funclet -t=dev
```

例如：打包runner-runtime

```$shell
$ sudo WITHNODEJS8=1 WITHNODEJS10=1 WITHNODEJS12=1 WITHPYTHON2=1 WITHPYTHON3=1 WITHJAVA8=1 bash scripts/runner-runtime/build.sh image:tag runnertag
```


您也可以提前配置不同的部署环境镜像仓库
eg: 
线上环境的配置目录 `~/zhangsan/docker/config/online`
测试环境的配置目录 `~/zhangsan/docker/config/test`

```$shell
 mkdir -p /tmp/docker/config/online /tmp/docker/config/test
 export DOCKER_CONFIG_ONLINE="/tmp/docker/config/online"
 export DOCKER_CONFIG_TEST="/tmp/docker/config/test"
 
 # online
 $ docker --config $DOCKER_CONFIG_ONLINE login registry.baidubce.com -u <online-user> -p'<online-password>'
 # test
 $ docker --config $DOCKER_CONFIG_TEST login registry.baidubce.com -u <test-user> -p'<test-password>'
```

## 3. 上传镜像

如果在本机，否则需要上传镜像

```$shell
docker push [OPTIONS] NAME[:TAG]
如：
docker push registry.baidubce.com/<your_namespace>/<your_image_name>:<your_image_tag>
```

## 4. 启动服务

```$shell
export faasPath=/<your_path_prefix>/openless/faas

#runner-runtime请直接下载镜像
docker run -t -d -e WITHRUNNER=1 -e WITHNODEJS10=1  -e WITHNODEJS12=1 -e WITHPYTHON3=1 --name runner-runtime -v ${faasPath}/runtime:/var/faas/runtime -v ${faasPath}/runner:/var/faas/runner registry.baidubce.com/openless-public/runner-runtime:demo1.0


#分别启动controller、stubs(func-registry)、funclet组件
docker run -t -d --privileged -v ${faasPath}/runner:/var/faas/runner -v ${faasPath}/runtime:/var/faas/runtime -v ${faasPath}/data:/var/faas/runner-data -v ${faasPath}/invoker/run:/var/run/faas -v ${faasPath}/funcData:/var/faas/funcData --name mini-funclet-dev  registry.baidubce.com/<your_namespace>/<your_funclet_image_name>:<your_image_tag> 

docker run -t -d --network host -v ${faasPath}/invoker/run:/var/run/faas --name controller-dev registry.baidubce.com/<your_namespace>/<your_controller_image_name>:<your_image_tag> /controller --maxprocs=10 --port=8899 --repository-endpoint=http://127.0.0.1:8002 --repository-version=v1 --repository-auth-type=noauth --logtostderr=true --http-enhanced=true -v10

docker run -t -d --network host --name func-registry -v ${faasPath}/funcData:/var/faas/funcData registry.baidubce.com<your_namespace>/<your_func_registry_image_name>:<your_image_tag>

```

备注：更详细的参数，可以参考方式二中的docker-compose.yaml文件


## 5. 操作运行

[运行步骤](./op_func.md)

