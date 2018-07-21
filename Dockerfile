FROM golang:1.10

RUN go get golang.org/x/tools/cmd/goimports
RUN go get -u github.com/golang/lint/golint
RUN go get github.com/golang/mock/gomock
RUN go install github.com/golang/mock/mockgen



RUN curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - && \
  echo "deb https://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list && \
  apt-get update || true && \
  apt-get install -y apt-transport-https && \
  apt-get update && apt-get install -y yarn curl

RUN curl -sL https://deb.nodesource.com/setup_8.x | bash - && \
  apt-get install -y nodejs

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
	&& apt-get install -y unzip \
	&& unzip "$TERRAFORM_ZIP" \
	&& mv "terraform" "/usr/local/bin/terraform-${TERRAFORM_VERSION}" \
	&& ln -s "/usr/local/bin/terraform-${TERRAFORM_VERSION}" /usr/local/bin/terraform

ENV DEP_URL=https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64
ENV DEP_BIN=dep-linux-amd64
ENV DEP_SHA256SUM=31144e465e52ffbc0035248a10ddea61a09bf28b00784fd3fdd9882c8cbb2315

RUN curl -fsSLO "${DEP_URL}" \
    && echo "${DEP_SHA256SUM}  ${DEP_BIN}" | sha256sum -c - \
	&& chmod +x ${DEP_BIN} \
    && mv ${DEP_BIN} /usr/local/bin/dep-linux-amd64 \
    && ln -s /usr/local/bin/dep-linux-amd64 /usr/local/bin/dep

ENV PROJECTPATH=/go/src/github.com/replicatedhq/ship


WORKDIR $PROJECTPATH
CMD ["/bin/bash"]
