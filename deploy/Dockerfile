FROM alpine:latest
RUN apk add --no-cache ca-certificates curl git && update-ca-certificates
COPY ship /ship
ENV IN_CONTAINER 1
ENV NO_OPEN 1
LABEL "com.replicated.ship"="true"
WORKDIR /out
ENTRYPOINT [ "/ship" ]
