FROM golang:1.10

RUN go get golang.org/x/tools/cmd/goimports
RUN go get -u github.com/golang/lint/golint
RUN go get github.com/golang/mock/gomock
RUN go install github.com/golang/mock/mockgen
RUN go get github.com/elazarl/go-bindata-assetfs/...
RUN go get -u github.com/jteeuwen/go-bindata/...


RUN curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - && \
  echo "deb https://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list && \
  apt-get update || true && \
  apt-get install -y apt-transport-https && \
  apt-get update && apt-get install -y yarn curl bzip2

RUN curl -sL https://deb.nodesource.com/setup_8.x | bash - && \
  apt-get install -y nodejs

ENV TERRAFORM_VERSION=0.11.14
ENV TERRAFORM_URL="https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip"
ENV TERRAFORM_ZIP="terraform_${TERRAFORM_VERSION}_linux_amd64.zip"
ENV TERRAFORM_SHA256SUM=9b9a4492738c69077b079e595f5b2a9ef1bc4e8fb5596610f69a6f322a8af8dd

RUN curl -fsSLO "$TERRAFORM_URL" \
	&& echo "${TERRAFORM_SHA256SUM}  ${TERRAFORM_ZIP}" | sha256sum -c - \
	&& apt-get install -y unzip \
	&& unzip "$TERRAFORM_ZIP" \
	&& mv "terraform" "/usr/local/bin/terraform-${TERRAFORM_VERSION}" \
	&& ln -s "/usr/local/bin/terraform-${TERRAFORM_VERSION}" /usr/local/bin/terraform

ENV DEP_URL=https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64
ENV DEP_BIN=dep-linux-amd64
ENV DEP_SHA256SUM=287b08291e14f1fae8ba44374b26a2b12eb941af3497ed0ca649253e21ba2f83

RUN curl -fsSLO "${DEP_URL}" \
    && echo "${DEP_SHA256SUM}  ${DEP_BIN}" | sha256sum -c - \
	&& chmod +x ${DEP_BIN} \
    && mv ${DEP_BIN} /usr/local/bin/dep-linux-amd64 \
    && ln -s /usr/local/bin/dep-linux-amd64 /usr/local/bin/dep

ENV PROJECTPATH=/go/src/github.com/replicatedhq/ship


WORKDIR $PROJECTPATH
CMD ["/bin/bash"]
