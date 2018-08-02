# build the test
FROM golang:1.10-alpine as build-step
ENV GOPATH=/go

RUN apk update && apk add ca-certificates curl git

ENV HELM_VERSION=v2.9.1
ENV HELM_URL=https://storage.googleapis.com/kubernetes-helm/helm-v2.9.1-linux-amd64.tar.gz
ENV HELM_TGZ=helm-v2.9.1-linux-amd64.tar.gz
ENV HELM=linux-amd64/helm
ENV HELM_SHA256SUM=56ae2d5d08c68d6e7400d462d6ed10c929effac929fedce18d2636a9b4e166ba

RUN curl -fsSLO "${HELM_URL}" \
    && echo "${HELM_SHA256SUM}  ${HELM_TGZ}" | sha256sum -c - \
    && tar xvf "$HELM_TGZ" \
    && mv "$HELM" "/usr/local/bin/helm-${HELM_VERSION}" \
    && ln -s "/usr/local/bin/helm-${HELM_VERSION}" /usr/local/bin/helm

ENV TERRAFORM_VERSION=0.11.7
ENV TERRAFORM_URL="https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip"
ENV TERRAFORM_ZIP="terraform_${TERRAFORM_VERSION}_linux_amd64.zip"
ENV TERRAFORM_SHA256SUM=6b8ce67647a59b2a3f70199c304abca0ddec0e49fd060944c26f666298e23418

RUN curl -fsSLO "$TERRAFORM_URL" \
	&& echo "${TERRAFORM_SHA256SUM}  ${TERRAFORM_ZIP}" | sha256sum -c - \
	&& unzip "$TERRAFORM_ZIP" \
	&& mv "terraform" "/usr/local/bin/terraform-${TERRAFORM_VERSION}" \
	&& ln -s "/usr/local/bin/terraform-${TERRAFORM_VERSION}" /usr/local/bin/terraform

RUN go get github.com/docker/distribution/cmd/registry

ADD . /go/src/github.com/replicatedhq/ship
RUN cd /go/src/github.com/replicatedhq/ship && go test -c ./integration/base && go test -c ./integration/update

# package things up
FROM alpine
ENV GOPATH=/go
WORKDIR /test

COPY --from=build-step /usr/local/bin/helm /usr/local/bin/helm
COPY --from=build-step /usr/local/bin/terraform /usr/local/bin/terraform
COPY --from=build-step $GOPATH/bin/registry $GOPATH/bin/registry
COPY --from=build-step $GOPATH/src/github.com/docker/distribution/cmd/registry/config-example.yml config.yml
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

RUN mkdir -p /var/lib/registry


ADD ./integration/base /test
ADD ./integration/update /test
RUN cd /test && rm *.go
COPY --from=build-step /go/src/github.com/replicatedhq/ship/base.test /test/
COPY --from=build-step /go/src/github.com/replicatedhq/ship/update.test /test/
ENTRYPOINT ./integration_docker_start.sh




