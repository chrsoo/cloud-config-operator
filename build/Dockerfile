FROM alpine:3.8

RUN apk upgrade --update --no-cache

# Latest version: https://storage.googleapis.com/kubernetes-release/release/stable.txt
# ADD https://storage.googleapis.com/kubernetes-release/release/stable.txt /usr/local/kubernetes.version
ARG kubernetes_version="v1.13.1"
ADD https://storage.googleapis.com/kubernetes-release/release/${kubernetes_version}/bin/linux/amd64/kubectl /usr/local/bin/kubectl
RUN chmod +x /usr/local/bin/kubectl \
    && kubectl version --client

USER nobody
ADD build/_output/bin/cloud-config-operator /usr/local/bin/cloud-config-operator
