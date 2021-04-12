FROM registry.baidubce.com/easyfaas-public/ubuntu:funclet-base
MAINTAINER fangliying@baidu.com

COPY funclet /funclet
COPY controller /controller
COPY stubs /stubs
COPY entrypoint.sh /entrypoint.sh

CMD ["/entrypoint.sh"]