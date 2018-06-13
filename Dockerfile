FROM golang:1.9

RUN go get golang.org/x/tools/cmd/goimports
RUN go get -u github.com/golang/lint/golint
RUN go get -u github.com/golang/dep/cmd/dep
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

ENV PROJECTPATH=/go/src/github.com/replicatedhq/ship


WORKDIR $PROJECTPATH
CMD ["/bin/bash"]
