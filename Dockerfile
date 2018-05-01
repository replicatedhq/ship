FROM golang:1.9

RUN go get golang.org/x/tools/cmd/goimports
RUN go get -u github.com/golang/lint/golint
RUN go get -u github.com/golang/dep/cmd/dep
RUN go get github.com/golang/mock/gomock
RUN go install github.com/golang/mock/mockgen

ENV PROJECTPATH=/go/src/github.com/replicatedcom/ship

WORKDIR $PROJECTPATH
CMD ["/bin/bash"]
