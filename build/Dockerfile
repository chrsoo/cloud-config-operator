FROM alpine:3.8

RUN apk upgrade --update --no-cache

USER nobody

ADD build/_output/bin/cloud-config-operator /usr/local/bin/cloud-config-operator
