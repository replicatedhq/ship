# build the test
FROM golang:1.10-alpine as build-step
ENV GOPATH=/go/
ADD . /go/src/github.com/replicatedhq/ship
RUN cd /go/src/github.com/replicatedhq/ship && go test -c ./integration

# package things up
FROM alpine
WORKDIR /test
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
ADD ./integration /test
RUN cd /test && rm *.go
COPY --from=build-step /go/src/github.com/replicatedhq/ship/integration.test /test/
ENTRYPOINT ./integration.test

