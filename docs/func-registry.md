# controller + stubs使用

## 1. 启动

此处，对于其余模块启动不赘述。详见README。

### stubs

启动参数
eg: function-dir设置为 /var/faas/funcData

```shell
./stubs --function-dir="/var/faas/funcData" --logtostderr --port 8002
```

### controller

启动参数

```shell
./controller --port=8001 --logtostderr --repository-endpoint=http://127.0.0.1:8002 --repository-version="v1" --repository-auth-type="noauth" --repository-auth-params="{}" --max-runtime-idle=10 --enable-metrics -v10
```



## 2. 使用stubs构建本地代码仓库

### 2.1 利用stubs创建函数

#### 请求结构
```shell
POST /v1/functions/<functionName> HTTP/1.1
base64_code_bytes
```

#### 参数
|参数名称|类型|是否必需|参数位置|描述|
|---|---|---|---|---|
|functionName|string|是|Path参数|函数名称，您可以指定一个函数名(例如，Thumbnail)，或者您可以指定函数的BRN资源名(例如，brn:bce:faas:bj:account-id:function:thumbnail:$LATEST)。faas也允许您指定一个部分的BRN(例如，account-id:Thumbnail)。注意，BRN长度限制为1-170。如果只指定函数名，则长度限制为64个字符|
|Version|string|否|Body参数|函数版本，若functionName指定为brn且指定了版本，此参数会被忽略。默认值为$LATEST|
|Description|string|否|Body参数|函数描述|
|Runtime|string|否|Body参数|函数运行时，现支持nodejs6.11/nodejs8.5/nodejs10/lua5.3，默认值为nodejs8.5|
|Timeout|int|否|Body参数|函数超时时间，默认值为5|
|MemorySize|int|否|Body参数|函数运行内存，默认值为128|
|Handler|string|否|Body参数|调用的入口函数，默认值为index.handler|
|PodConcurrentQuota|int|否|Body参数|预留并发度，默认值为0|
|Code|string|否|Body参数|函数代码zip包的base64字符串，默认为一个nodejs的hello world函数|

#### 请求示例
```shell
POST /v1/functions/testHelloWorld HTTP/1.1
Content-Type: application/json; charset=utf-8
Host: 127.0.0.1:8002
Connection: close
User-Agent: Paw/3.1.8 (Macintosh; OS X/10.15.2) GCDHTTPRequest
Content-Length: 990

{
    "Version":"1",
    "Description":"stubs create",
    "Runtime":"python2.7",
    "Timeout":20,
    "MemorySize":512,
    "Handler":"index.hanler",
    "PodConcurrentQuota":10,
    "Code":"UEsDBBQACAAIAAyjX00AAAAAAAAAAAAAAAAIABAAaW5kZXguanNVWAwAsJ/ZW/ie2Vv6Z7qeS60o\nyC8qKdbLSMxLyUktUrBV0EgtS80r0VFIzs8rSa0AMRJzcpISk7M1FWztFKq5FIAAJqSRV5qTo6Og\n5JGak5OvUJ5flJOiqKRpzVVrDQBQSwcILzRMjVAAAABYAAAAUEsDBAoAAAAAAHCjX00AAAAAAAAA\nAAAAAAAJABAAX19NQUNPU1gvVVgMALSf2Vu0n9lb+me6nlBLAwQUAAgACAAMo19NAAAAAAAAAAAA\nAAAAEwAQAF9fTUFDT1NYLy5faW5kZXguanNVWAwAsJ/ZW/ie2Vv6Z7qeY2AVY2dgYmDwTUxW8A9W\niFCAApAYAycQGwFxHRCD+BsYiAKOISFBUCZIxwIgFkBTwogQl0rOz9VLLCjISdXLSSwuKS1OTUlJ\nLElVDggGKXw772Y0iO5J8tAH0QBQSwcIDgnJLFwAAACwAAAAUEsBAhUDFAAIAAgADKNfTS80TI1Q\nAAAAWAAAAAgADAAAAAAAAAAAQKSBAAAAAGluZGV4LmpzVVgIALCf2Vv4ntlbUEsBAhUDCgAAAAAA\ncKNfTQAAAAAAAAAAAAAAAAkADAAAAAAAAAAAQP1BlgAAAF9fTUFDT1NYL1VYCAC0n9lbtJ/ZW1BL\nAQIVAxQACAAIAAyjX00OCcksXAAAALAAAAATAAwAAAAAAAAAAECkgc0AAABfX01BQ09TWC8uX2lu\nZGV4LmpzVVgIALCf2Vv4ntlbUEsFBgAAAAADAAMA0gAAAHoBAAAAAA=="
}
```
#### 响应示例
```shell
HTTP/1.1 200 OK
Server: fasthttp
```



### 2.2 本地代码仓结构

```shell
$ tree /var/faas/funcData
/var/faas/funcData
└── brn:bce:faas:bj:cd64f99c69d7c404b61de0a4f1865834:function:testHelloWorld:1 // 函数brn
  ├── code.zip // 代码zip包
  └── meta.json // 函数元信息
```

### function meta

```json
{
  "_": {},
  "Code": {
    "_": {},
    "Location": "/var/faas/funcData/brn:cloud:faas:bj:cd64f99c69d7c404b61de0a4f1865834:function:testHelloWorld:1/code.zip",
    "RepositoryType": "filesystem"
  },
  "Concurrency": {
    "_": {},
    "ReservedConcurrentExecutions": null,
    "AccountReservedSum": 0
  },
  "Configuration": {
    "_": {},
    "CodeSha256": "4OFxEke82hUugwILdGb/BxnQdSUTsPAYcSU9PNVdFlU=",
    "CodeSize": 610,
    "DeadLetterConfig": null,
    "Description": "stubs create",
    "Environment": {
      "_": {},
      "Error": null,
      "Variables": null
    },
    "FunctionArn": "brn:cloud:faas:bj:cd64f99c69d7c404b61de0a4f1865834:function:testHelloWorld:1",
    "FunctionName": "testHelloWorld",
    "Handler": "index.hanler",
    "KMSKeyArn": null,
    "LastModified": "2020-01-06T14:04:19+08:00",
    "Layers": null,
    "MasterArn": null,
    "MemorySize": 512,
    "RevisionId": null,
    "Role": null,
    "Runtime": "python2.7",
    "Timeout": 20,
    "TracingConfig": null,
    "Version": "1",
    "VpcConfig": null,
    "CommitId": "349dd8a4-db7d-4fc3-a4cd-8c8b482e00c3",
    "Uid": "df391b08c64c426a81645468c75163a5",
    "SourceTag": "",
    "DeadLetterTopic": "",
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

### function code

```javascript
  exports.handler = (event, context, callback) => {
  callback(null, "Hello world!WooHoo");
  };
```

### 3. 测试

测试stubs

```shell
$ curl http://127.0.0.1:8002/v1/functions/brn:bce:faas:bj:cd64f99c69d7c404b61de0a4f1865834:function:testHelloWorld:1
```

controller invoke一个请求

```shell
$ curl -X "POST" "http://127.0.0.1:8001/v1/functions/brn:bce:faas:bj:cd64f99c69d7c404b61de0a4f1865834:function:testHelloWorld:1/invocations"   -H 'X-Bce-Account-Id: df391b08c64c426a81645468c75163a5'      -H 'Content-Type: application/json; charset=utf-8'
```
