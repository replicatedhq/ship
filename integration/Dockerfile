# build the test
FROM golang:1.10-alpine as build-step
ENV GOPATH=/go/
ADD . /go/src/github.com/replicatedhq/ship
RUN cd /go/src/github.com/replicatedhq/ship && go test -c ./integration

RUN apk update && apk add ca-certificates && apk add curl

ENV HELM_VERSION=v2.9.1
ENV HELM_URL=https://storage.googleapis.com/kubernetes-helm/helm-v2.9.1-linux-amd64.tar.gz
ENV HELM_TGZ=helm-v2.9.1-linux-amd64.tar.gz
ENV HELM=linux-amd64/helm
ENV HELM_SHA256SUM=56ae2d5d08c68d6e7400d462d6ed10c929effac929fedce18d2636a9b4e166ba

RUN curl -fsSLO "${HELM_URL}" \
    && echo "${HELM_SHA256SUM}  ${HELM_TGZ}" | sha256sum -c - \
    && tar xvf "$HELM_TGZ" \
    && mv "$HELM" "/usr/local/bin/helm-${HELM_VERSION}" \
    && cp "/usr/local/bin/helm-${HELM_VERSION}" /usr/local/bin/helm

# package things up
FROM alpine
WORKDIR /test

COPY --from=build-step /usr/local/bin/helm /usr/local/bin/helm
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

ADD ./integration /test
RUN cd /test && rm *.go
COPY --from=build-step /go/src/github.com/replicatedhq/ship/integration.test /test/
ENTRYPOINT ./integration.test

