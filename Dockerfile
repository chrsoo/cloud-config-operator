FROM chrsoo/operator-sdk AS builder

ENV NAMESPACE=chrsoo
ENV NAME=cloud-config-operator

WORKDIR /go/src/github.com/${NAMESPACE}/${NAME}
COPY . /go/src/github.com/${NAMESPACE}/${NAME}
RUN go build $GOFLAGS -o /go/bin/${NAME} github.com/${NAMESPACE}/${NAME}/cmd/manager

# Base image
FROM registry.access.redhat.com/ubi7-dev-preview/ubi-minimal:7.6

ENV OPERATOR=/usr/local/bin/${NAME} \
    USER_UID=1001 \
    USER_NAME=${NAME}

# install operator binary
COPY --from=builder /go/bin/${NAME} ${OPERATOR}
# COPY build/bin /usr/local/bin

# RUN  /usr/local/bin/user_setup
# ENTRYPOINT ["/usr/local/bin/entrypoint"]
# USER ${USER_UID}
