FROM alpine:3.6
MAINTAINER Rohith Jayawardene <gambol99@gmail.com>

RUN apk add ca-certificates --update

ADD bin/kube-cert-ingress /kube-cert-ingress

RUN adduser -D controller
USER controller

ENTRYPOINT [ "/kube-cert-ingress" ]
