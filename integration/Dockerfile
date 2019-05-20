# build the test
FROM golang:1.12-alpine as build-step
ENV GOPATH=/go

RUN apk update && apk add ca-certificates curl git build-base

ENV TERRAFORM_VERSION=0.11.14
ENV TERRAFORM_URL="https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip"
ENV TERRAFORM_ZIP="terraform_${TERRAFORM_VERSION}_linux_amd64.zip"
ENV TERRAFORM_SHA256SUM=9b9a4492738c69077b079e595f5b2a9ef1bc4e8fb5596610f69a6f322a8af8dd

RUN curl -fsSLO "$TERRAFORM_URL" \
	&& echo "${TERRAFORM_SHA256SUM}  ${TERRAFORM_ZIP}" | sha256sum -c - \
	&& unzip "$TERRAFORM_ZIP" \
	&& mv "terraform" "/usr/local/bin/terraform-${TERRAFORM_VERSION}" \
	&& ln -s "/usr/local/bin/terraform-${TERRAFORM_VERSION}" /usr/local/bin/terraform

ENV KUBECTL_VERSION=v1.11.1
ENV KUBECTL_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_SHA256SUM=d16a4e7bfe0033ea5f56f8d11e74f7a2dec5ff8832a046a643c8355b79b4ba5c

RUN curl -fsSLO "${KUBECTL_URL}" \
	&& echo "${KUBECTL_SHA256SUM}  kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_VERSION}" \
	&& ln -s "/usr/local/bin/kubectl-${KUBECTL_VERSION}" /usr/local/bin/kubectl

RUN go get github.com/docker/distribution/cmd/registry
RUN go get github.com/onsi/ginkgo/ginkgo

ADD . /go/src/github.com/replicatedhq/ship
RUN cd /go/src/github.com/replicatedhq/ship && \
    ginkgo build ./integration/base && \
    ginkgo build ./integration/update && \
    ginkgo build ./integration/init_app && \
    ginkgo build ./integration/init && \
    ginkgo build ./integration/unfork

# package things up
FROM node:8-alpine
ENV GOPATH=/go
WORKDIR /test

RUN npm install -g http-echo-server

COPY --from=build-step /usr/local/bin/terraform /usr/local/bin/terraform
COPY --from=build-step /usr/local/bin/kubectl /usr/local/bin/kubectl
COPY --from=build-step $GOPATH/bin/registry $GOPATH/bin/registry
COPY --from=build-step $GOPATH/bin/ginkgo $GOPATH/bin/ginkgo
RUN apk update && apk add ca-certificates git openssh && rm -rf /var/cache/apk/*

RUN mkdir -p /var/lib/registry


ADD ./integration /test
RUN cd /test && rm *.go
COPY --from=build-step /go/src/github.com/replicatedhq/ship/integration/base/base.test /test/base/
COPY --from=build-step /go/src/github.com/replicatedhq/ship/integration/update/update.test /test/update/
COPY --from=build-step /go/src/github.com/replicatedhq/ship/integration/init_app/init_app.test /test/init_app/
COPY --from=build-step /go/src/github.com/replicatedhq/ship/integration/init/init.test /test/init/
COPY --from=build-step /go/src/github.com/replicatedhq/ship/integration/unfork/unfork.test /test/unfork
CMD ./integration_docker_start.sh




