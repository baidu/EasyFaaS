# 质量测试

**函数调用测试**

```
$ curl -X "POST" "http://127.0.0.1:8001/v1/functions/brn:bce:faas:bj:cd64f99c69d7c404b61de0a4f1865834:function:testHelloWorld:%24LATEST/invocations"   -H 'X-Bce-Account-Id: df391b08c64c426a81645468c75163a5'      -H 'Content-Type: application/json; charset=utf-8'      -d $'{}'
```

**压力测试**

```shell
demo（您也可以选用如ab的压测工具进行）
$./cli bench "http://127.0.0.1:8001/v1/functions/brn:bce:faas:bj:cd64f99c69d7c404b61de0a4f1865834:function:testHelloWorld:%24LATEST/invocations" -m POST --ak=1 --sk=1 --headers 'X-Bce-Request-Id: 0c5d77c5-4440-41fb-92b3-8fcb3bc30c15' --headers "X-Bce-Account-Id: df391b08c64c426a81645468c75163a5" --headers 'Content-Type: application/json; charset=utf-8' -d '' -c 3 -n 1000
```

