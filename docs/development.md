# 开发手册
基于Linux（内核4.0+）、提前安装docker，再进行下面步骤：

## 1. 代码目录数结构
```shell
├── Makefile                         # Makefile
├── README.md                        # ReadMe文档
├── build                            # 构建脚本目录
│   ├── common.sh                    # 构建项目的全局变量设置
│   ├── lib
│   │   ├── golang.sh                # 项目编译的基础设置脚本,如编译平台等
│   │   ├── init.sh                  # 编译构建脚本的初始化设置，如导入其他构建lib
│   │   ├── logging.sh               # 编译构建脚本的日志功能，如错误日志等
│   │   ├── util.sh                  # 编译构建脚本的工具函数,如获取本机arch等
│   │   └── version.sh               # 二进制的版本管理lib 
│   ├── make-clean.sh                # 清理构建产出
│   └── make-rules                       
│       ├── build.sh                 # 编译构建各组件二进制
│       └── test.sh                  # 运行项目单测及汇报覆盖率
├── build.sh                         # 构建脚本
├── cmd                              # 所有模块入口
│   ├── controller                   # controller组件
│   ├── funclet                      # funclet组件
│   ├── httptrigger                  # http触发器组件
│   └── stubs                        # local-registry stub组件
├── docs                             # 文档目录
│   ├── func-registry.md             # stubs生成func-registry文档                        
│   ├── development.md               # 本地开发文档
│   ├── getting-started.md           # 使用docker-compose快速体验easyfaas服务文档
│   └── test.md                      # 测试文档
├── go.mod
├── go.sum
├── pkg
│   ├── api                          # 各模块API结构体及常量包
│   ├── auth                         # 鉴权包,目前支持百度云鉴权\不鉴权两种方式，注册自定义鉴权方式
│   ├── brn                          # brn包，提供对资源brn的解析
│   ├── controller                   # controller组件：实现了用户流量调度以及容器池状态管理等功能
│   ├── error                        # error包：定义通用easyfaas服务通用error
│   ├── funclet                      # funclet组件：实现用户工作容器的管理
│   ├── httptrigger                  # httptrigger组件：支持http请求触发函数调用
│   ├── repository                   # 函数代码存储组件，可注册自定义代码存储服务
│   ├── rest                         # rest包：rest http client 服务，用于发起http请求
│   ├── server                       # http server包：包含server的配置，过滤，hook等功能
│   ├── stubs                        # local-registry stub组件：提供本地函数代码存储服务
│   ├── userlog                      # 用户日志：用户函数日志的格式化及收集，支持plain和json格式
│   ├── util                         # util包：包含本地缓存/挂载/文件等工具函数
│   └── version                      # version包：用于获取组件二进制的版本
└── scripts                          # 工具脚本
    ├── build-image.sh               # 镜像制作脚本
    ├── controller                   # 制作controller镜像时所需文件                    
    ├── funclet                      # 制作funclet镜像时所需文件   
    ├── runner-runtime               # 制作runner-runtine镜像时所需文件
    └── stubs                        # 制作local-registry stubs 镜像时所需文件
    └── docker-compose               # 快速上手使用docker-compose跑easyfaas服务所需脚本
```

## 2. 编译
2.1 编译全部模块
```$shell
$ make
```
2.2 为其它平台交叉编译：
```bash
$ KUN_BUILD_PLATFORMS="linux/amd64" make
```

2.3 只编译某一个模块：
```bash
$ make WHAT=./pkg/controller
```

## 3. 打包

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

例如：打包 controller/funclet
```$shell
$ scripts/build-image.sh -e=test -m=funclet -i=mini-funclet -t=dev
```

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


## 4 调试
### 4.1 环境变量
在宿主机上准备faas的数据目录`${faasPath}`，eg: `/home/faas`
```shell
export faasPath=/home/faas
# 本地bin包目录
export faasDevBin=/home/xflying/controller/code
```

### 4.2 准备runtime及runner
准备runtime及runner
```shell
$ docker run  -t -d --name runner-runtime -v ${faasPath}/runtime:/var/faas/runtime -v ${faasPath}/runner:/var/faas/runner registry.baidubce.com/easyfaas-public/runner-runtime:ba91af

# 内存不够的开发人员，可以在容器准备完毕数据后，stop掉容器
# 判定数据准备完毕：查看runtime日志，直到看到循环sleep为止
$ docker logs runner-runtime  
$ docker stop runner-runtime
```

### 4.3 启动或调试
#### 4.3.1 调试funclet
若不需要更改funclet，并只想以默认参数启动funclet，则
```shell
$ docker run -t -d --privileged -v ${faasPath}:/var/faas -v ${faasPath}/invoker/run:/var/run/faas --name mini-funclet registry.baidubce.com/easyfaas-public/mini-funclet:907546
```

若需要更改funclet，则以bash作为init命令即可，随后进入容器进行调试
ps : 可将自己的代码mount进容器，此处更换掉`/home/fly/funclet`即可
```shell
$ docker run -t -d --privileged -v ${faasDevBin}:/home/code -v ${faasPath}:/var/faas -v ${faasPath}/invoker/run:/var/run/faas --name mini-funclet-dev  registry.baidubce.com/easyfaas-public/mini-funclet:82b366 /bin/bash
$ docker exec -it mini-funclet-dev /bin/bash
$ ./funclet -v10 --logtostderr
```

#### 4.3.2 调试controller
若不需要更改controller，并只想以默认参数启动controller，则
```shell
$ docker run -t -d -v ${faasPath}/invoker/run:/var/run/faas --name controller  regisry.baidubce.com/easyfaas-public/controller:28a179
```

若需要更改controller，则以sh作为init命令即可，随后进入容器进行调试
ps : 可将自己的代码mount进容器，此处更换掉`/home/fly/controller`即可

```shell
$ docker run -t -d -v ${faasDevBin}:/home/code -v ${faasPath}/invoker/run:/var/run/faas --name controller-dev registry.baidubce.com/easyfaas-public/controller:28a179 sh
$ docker exec -it controller-dev sh
$ ./controller --port=8001 --logtostderr --repository-endpoint=http://cfc.bj.baidubce.com --repository-version="v1/ote" --repository-auth-type="bce" --repository-auth-params="{\"ak\":\"1e5abd9730754bcfb6aab8e3f1c3e6c7\", \"sk\": \"1a8d187b96f04699af506afd7e8a9048\"}" --enable-metrics -v10
```

