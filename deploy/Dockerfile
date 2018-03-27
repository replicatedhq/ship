FROM debian:stretch-slim
RUN DEBIAN_FRONTEND=noninteractive apt-get update && apt-get install --no-install-recommends -y \
    ca-certificates \
    procps \
    iproute2 \
    sensible-utils \
    systemd \
    libc6 \
    libc-bin \
    multiarch-support \
    libudev1 \
    libncurses5 \
    ncurses-base \
    ncurses-bin \
    libblkid1 \
    libfdisk1 \
    libmount1 \
    libsmartcols1 \
    libuuid1 \
    mount \
    util-linux \
  && rm -rf /var/lib/apt/lists/*
COPY ship /usr/local/bin/ship
ENV IN_CONTAINER 1
LABEL "com.replicated.ship"="true"
ENTRYPOINT [ "ship" ]
CMD [ "apply" ]
