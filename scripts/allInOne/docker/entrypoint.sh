#!/bin/bash

set -x
set -e

/stubs --port=8002 --function-dir=/var/faas/funcData --log-dir=/var/log --log-level=info &
/funclet --reserved-memory=100M --reserved-cpu=0.3 --runner-memory=128M --container-num=20 --log-dir=/var/log --log-level=info -v10 &
/controller --maxprocs=6 --port=8001 --repository-endpoint=http://127.0.0.1:8002 --repository-version=v1 --repository-auth-type=noauth --log-dir=/var/log --log-level=info --http-enhanced=true -v10