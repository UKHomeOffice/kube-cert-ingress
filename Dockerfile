FROM alpine:3.7
MAINTAINER Rohith Jayawardene <gambol99@gmail.com>

RUN apk add ca-certificates --update

ADD bin/kube-cert-ingress /kube-cert-ingress

RUN adduser -D -u 1000 controller
USER 1000

ENTRYPOINT [ "/kube-cert-ingress" ]
