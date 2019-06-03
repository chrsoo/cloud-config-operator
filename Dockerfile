FROM chrsoo/operator-sdk:latest AS builder

# Operator base image
FROM chrsoo/operator-base:latest
ARG kubernetes_version="v1.13.1"
ADD https://storage.googleapis.com/kubernetes-release/release/${kubernetes_version}/bin/linux/amd64/kubectl /usr/local/bin/kubectl
