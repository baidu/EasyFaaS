# 快速开始
基于Linux（内核4.0+）、提前安装docker, 提供两种方式供开发者快速开始：


- 方式一：使用all-in-one方式运行
- 方式二：使用docker-compose运行
- 方式三：编译打包部署



## 方式一：使用all-in-one方式运行

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

打包all-in-one镜像，其中会包含controller组件、 funclet组件、 stubs组件。需要注意的runtime目前不开源，直接拉取镜像即可，执行脚本打包：

```$shell
$ scripts/build-image.sh -m=one -i=easyfaas -t=dev
```

如果您需要发布打包好的镜像，需要指定REGISTRY，如：

```$shell
$REGISTRY=registry.baidubce.com/<your_namespace>/ ./scripts/build-image.sh -m=one -i=easyfaas -t=dev
```

REGISTRY地址可以在 [百度云容器镜像服务](https://cloud.baidu.com/doc/CCR/s/qk8gwqs4a) 中免费申请体验

## 3. 上传镜像

如果在本机，否则需要上传镜像

```$shell
docker push [OPTIONS] NAME[:TAG]
如：
docker push registry.baidubce.com/<your_namespace>/<your_image_name>:<your_image_tag>
```

## 4. 启动服务

```$shell
export faasPath=/<your_path_prefix>/easyfaas/faas

#runner-runtime请直接下载镜像
docker run -td -e WITHRUNNER=1 -e WITHNODEJS10=1  -e WITHNODEJS12=1 -e WITHPYTHON3=1 --name runner-runtime -v ${faasPath}/runtime:/var/faas/runtime -v ${faasPath}/runner:/var/faas/runner registry.baidubce.com/easyfaas-public/runner-runtime:demo1.0

# 启动all-in-one服务
docker run -td --privileged -v ${faasPath}/runner:/var/faas/runner -v ${faasPath}/runtime:/var/faas/runtime -v ${faasPath}/data:/var/faas/runner-data --name easyfaas registry.baidubce.com/easyfaas/all-in-one:demo1.0
```

## 5. 操作运行

安装运行完成后，执行如下命令进入到容器内：

```shell
$ docker exec -it easyfaas bash
```

### 1. 创建函数

使用stub组件构建本地代码仓库其创建函数接口API详见接口API文档, 可以使用curl或者其他工具如Paw发起创建函数请求。


#### 1.1 编写函数代码  

首先编写一个简单的nodejs10的hello world 函数代码,代码存储在index.js文件中
备注：
您也可以直接跳过此步骤，直接运行1.1和1.2，直接运行1.3

```
$ cat index.js 
exports.handler = (event, context, callback) => {
    callback(null, "Hello world!");
};

```

#### 1.2 打包函数代码并获得其base64编码 

```cassandraql
$ zip code.zip index.js 
$ base64 code.zip 
UEsDBBQAAAAIAAxDX00vNEyNUAAAAFgAAAAIABwAaW5kZXguanNVVAkAA/ie2VuKQthfdXgLAAEE
6AMAAAToAwAAS60oyC8qKdbLSMxLyUktUrBV0EgtS80r0VFIzs8rSa0AMRJzcpISk7M1FWztFKq5
FIAAJqSRV5qTo6Og5JGak5OvUJ5flJOiqKRpzVVrDQBQSwECHgMUAAAACAAMQ19NLzRMjVAAAABY
AAAACAAYAAAAAAABAAAApIEAAAAAaW5kZXguanNVVAUAA/ie2Vt1eAsAAQToAwAABOgDAABQSwUG
AAAAAAEAAQBOAAAAkgAAAAAA

```

#### 1.3 调用func-registry创建函数接口创建函数

请求body中Code字段填入上一步骤中获得的base64编码

```cassandraql
$ curl -X POST "http://127.0.0.1:8002/v1/functions/testHelloWorld" -d '{"Version":"1","Description":"stubs create","Runtime":"nodejs10","Timeout":5,"MemorySize":128,"Handler":"index.handler","PodConcurrentQuota":10,"Code":"UEsDBBQAAAAAAHCjX00AAAAAAAAAAAAAAAAJABUAX19NQUNPU1gvVVgIALSf2Vu0n9lbVVQFAAG0n9lbUEsDBBQACAAIAAyjX00AAAAAAAAAAAAAAAATABUAX19NQUNPU1gvLl9pbmRleC5qc1VYCACwn9lb+J7ZW1VUBQAB+J7ZW2JgFWNnYGJg8E1MVvAPVohQgAKQGAMnAwODEQMDQx0DA5i/gYEo4BgSEgRlgnQsYGBgEEBTwogQl0rOz9VLLCjISdXLSSwuKS1OTUlJLElVDggGKXw772Y0iO5J8tAH0YAAAAD//1BLBwgOCcksZgAAALAAAABQSwMEFAAIAAgAAAAAAAAAAAAAAAAAAAAAAAgAAABpbmRleC5qc0qtKMgvKinWy0jMS8lJLVKwVdBILUvNK9FRSM7PK0mtADESc3KSEpOzNRVs7RSquRQUFOBCGnmlOTk6CkoeqTk5+Qrl+UU5KYpKmtZctdaAAAAA//9QSwcILzRMjVUAAABYAAAAUEsBAhQDFAAAAAAAcKNfTQAAAAAAAAAAAAAAAAkAFQAAAAAAAAAAQP1BAAAAAF9fTUFDT1NYL1VYCAC0n9lbtJ/ZW1VUBQABtJ/ZW1BLAQIUAxQACAAIAAyjX00OCcksZgAAALAAAAATABUAAAAAAAAAAECkgTwAAABfX01BQ09TWC8uX2luZGV4LmpzVVgIALCf2Vv4ntlbVVQFAAH4ntlbUEsBAhQAFAAIAAgAAAAAAC80TI1VAAAAWAAAAAgAAAAAAAAAAAAAAAAA+AAAAGluZGV4LmpzUEsFBgAAAAADAAMA2AAAAIMBAAAAAA=="}' -H 'X-easyfaas-Account-Id: df391b08c64c426a81645468c75163a5'   -H 'Content-Type: application/json; charset=utf-8'
请求示例
POST /v1/functions/testHelloWorld HTTP/1.1
Content-Type: application/json; charset=utf-8
Host: 172.22.170.33:8002
Connection: close
User-Agent: Paw/3.1.10 (Macintosh; OS X/10.15.4) GCDHTTPRequest
Content-Length: 990

{"Version":"1","Description":"stubs create","Runtime":"nodejs10","Timeout":5,"MemorySize":128,"Handler":"index.handler","PodConcurrentQuota":10,"Code":"UEsDBBQAAAAAAHCjX00AAAAAAAAAAAAAAAAJABUAX19NQUNPU1gvVVgIALSf2Vu0n9lbVVQFAAG0n9lbUEsDBBQACAAIAAyjX00AAAAAAAAAAAAAAAATABUAX19NQUNPU1gvLl9pbmRleC5qc1VYCACwn9lb+J7ZW1VUBQAB+J7ZW2JgFWNnYGJg8E1MVvAPVohQgAKQGAMnAwODEQMDQx0DA5i/gYEo4BgSEgRlgnQsYGBgEEBTwogQl0rOz9VLLCjISdXLSSwuKS1OTUlJLElVDggGKXw772Y0iO5J8tAH0YAAAAD//1BLBwgOCcksZgAAALAAAABQSwMEFAAIAAgAAAAAAAAAAAAAAAAAAAAAAAgAAABpbmRleC5qc0qtKMgvKinWy0jMS8lJLVKwVdBILUvNK9FRSM7PK0mtADESc3KSEpOzNRVs7RSquRQUFOBCGnmlOTk6CkoeqTk5+Qrl+UU5KYpKmtZctdaAAAAA//9QSwcILzRMjVUAAABYAAAAUEsBAhQDFAAAAAAAcKNfTQAAAAAAAAAAAAAAAAkAFQAAAAAAAAAAQP1BAAAAAF9fTUFDT1NYL1VYCAC0n9lbtJ/ZW1VUBQABtJ/ZW1BLAQIUAxQACAAIAAyjX00OCcksZgAAALAAAAATABUAAAAAAAAAAECkgTwAAABfX01BQ09TWC8uX2luZGV4LmpzVVgIALCf2Vv4ntlbVVQFAAH4ntlbUEsBAhQAFAAIAAgAAAAAAC80TI1VAAAAWAAAAAgAAAAAAAAAAAAAAAAA+AAAAGluZGV4LmpzUEsFBgAAAAADAAMA2AAAAIMBAAAAAA=="}

响应结果:
HTTP/1.1 200 OK
Server: fasthttp
Date: Fri, 11 Dec 2020 03:07:50 GMT
Content-Length: 0
Connection: close

```

### 2. 查看函数 

请求示例

```cassandraql
$ curl -X GET "http://127.0.0.1:8002/v1/functions/brn:cloud:faas:bj:8f6e5a28c663957ea04522547a66d08f:function:testHelloWorld:1" -H 'X-easyfaas-Account-Id: df391b08c64c426a81645468c75163a5'   -H 'Content-Type: application/json; charset=utf-8' 

GET /v1/functions/testHelloWorld HTTP/1.1
Host: 127.0.0.1:8002
Connection: close

```

响应结果

```json        
{
  "Code": {
    "_": {},
    "Location": "/var/faas/funcData/brn:cloud:faas:bj:8f6e5a28c663957ea04522547a66d08f:function:testHelloWorld:1/code.zip",
    "RepositoryType": "filesystem",
    "LogType": ""
  },
  "Concurrency": {
    "_": {},
    "ReservedConcurrentExecutions": null,
    "AccountReservedSum": 0
  },
  "Configuration": {
    "_": {},
    "CodeSha256": "VoLwl2uaH/cF0hsvKQMkLbbd7JKtGgG5MNExk8whT+M=",
    "CodeSize": 625,
    "DeadLetterConfig": null,
    "Description": "stubs create",
    "Environment": {
      "_": {},
      "Error": null,
      "Variables": null
    },
    "FunctionArn": "brn:cloud:faas:bj:8f6e5a28c663957ea04522547a66d08f:function:testHelloWorld:1",
    "FunctionName": "testHelloWorld",
    "Handler": "index.handler",
    "KMSKeyArn": null,
    "LastModified": "2021-02-28T11:11:07Z",
    "Layers": null,
    "MasterArn": null,
    "MemorySize": 128,
    "RevisionId": null,
    "Role": null,
    "Runtime": "nodejs10",
    "Timeout": 5,
    "TracingConfig": null,
    "Version": "1",
    "VpcConfig": null,
    "CommitId": "349dd8a4-db7d-4fc3-a4cd-8c8b482e00c3",
    "Uid": "df391b08c64c426a81645468c75163a5",
    "PodConcurrentQuota": 10
  },
  "LogConfig": {
    "LogType": "",
    "BosDir": "",
    "Params": ""
  },
  "Tags": {}
}
```

### 3. 运行函数

向controller发起函数调用请求

```cassandraql
$ curl -X "POST" "http://127.0.0.1:8001/v1/functions/brn:cloud:faas:bj:8f6e5a28c663957ea04522547a66d08f:function:testHelloWorld:1/invocations"   -H 'X-easyfaas-Account-Id: df391b08c64c426a81645468c75163a5'      -H 'Content-Type: application/json; charset=utf-8'      -d $'{}'
请求示例
POST /v1/functions/brn:bce:cfc:bj:cd64f99c69d7c404b61de0a4f1865834:function:testHSF2:1/invocations HTTP/1.1
Content-Type: application/json; charset=utf-8
X-easyfaas-Account-Id: df391b08c64c426a81645468c75163a5
Host: 127.0.0.1:8001
Connection: close

响应示例：
Hello world!
```

## 方式二 : 使用docker-compose运行

[运行步骤](./getting-started-docker-compose.md)

## 方式三：编译打包部署 

[运行步骤](./getting-started-docker.md)

