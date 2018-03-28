FROM alpine:latest
RUN apk add --no-cache ca-certificates && update-ca-certificates
COPY ship /ship
ENV IN_CONTAINER 1
LABEL "com.replicated.ship"="true"
WORKDIR /out
ENTRYPOINT [ "/ship" ]
CMD [ "apply" ]
