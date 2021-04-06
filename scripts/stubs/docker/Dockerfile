FROM registry.baidubce.com/openless-public/controller-base:v1.0
MAINTAINER fangliying@baidu.com

ADD stubs /stubs

CMD ["/stubs", "--function-dir=/var/faas/funcData", "--logtostderr", "--port=8002"]