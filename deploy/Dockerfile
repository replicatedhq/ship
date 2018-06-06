FROM alpine:latest
RUN apk add --no-cache ca-certificates curl && update-ca-certificates
COPY ship /ship
ENV IN_CONTAINER 1

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


LABEL "com.replicated.ship"="true"
WORKDIR /out
ENTRYPOINT [ "/ship" ]
