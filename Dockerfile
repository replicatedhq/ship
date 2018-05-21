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
  apt-get update && apt-get install -y yarn

RUN curl -sL https://deb.nodesource.com/setup_8.x | bash - && \
  apt-get install -y nodejs

ENV PROJECTPATH=/go/src/github.com/replicatedcom/ship

WORKDIR $PROJECTPATH
CMD ["/bin/bash"]
