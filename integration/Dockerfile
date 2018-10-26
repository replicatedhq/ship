# build the test
FROM golang:1.10-alpine as build-step
ENV GOPATH=/go

RUN apk update && apk add ca-certificates curl git

ENV TERRAFORM_VERSION=0.11.10
ENV TERRAFORM_URL="https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip"
ENV TERRAFORM_ZIP="terraform_${TERRAFORM_VERSION}_linux_amd64.zip"
ENV TERRAFORM_SHA256SUM=43543a0e56e31b0952ea3623521917e060f2718ab06fe2b2d506cfaa14d54527

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

ADD . /go/src/github.com/replicatedhq/ship
RUN cd /go/src/github.com/replicatedhq/ship && \
    go test -c ./integration/base && \
    go test -c ./integration/update && \
    go test -c ./integration/init_app && \
    go test -c ./integration/init

# package things up
FROM node:8-alpine
ENV GOPATH=/go
WORKDIR /test

RUN npm install -g http-echo-server

COPY --from=build-step /usr/local/bin/terraform /usr/local/bin/terraform
COPY --from=build-step /usr/local/bin/kubectl /usr/local/bin/kubectl
COPY --from=build-step $GOPATH/bin/registry $GOPATH/bin/registry
RUN apk update && apk add ca-certificates git openssh && rm -rf /var/cache/apk/*

RUN mkdir -p /var/lib/registry


ADD ./integration /test
RUN cd /test && rm *.go
COPY --from=build-step /go/src/github.com/replicatedhq/ship/base.test /test/base/
COPY --from=build-step /go/src/github.com/replicatedhq/ship/update.test /test/update/
COPY --from=build-step /go/src/github.com/replicatedhq/ship/init_app.test /test/init_app/
COPY --from=build-step /go/src/github.com/replicatedhq/ship/init.test /test/init/
CMD ./integration_docker_start.sh




