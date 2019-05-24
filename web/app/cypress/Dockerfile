# Build Ship
FROM avcosystems/golang-node as build-step
ENV GOPATH=/go
RUN apt-get install bzip2
ADD . /go/src/github.com/replicatedhq/ship
WORKDIR /go/src/github.com/replicatedhq/ship
RUN make build-ci-cypress

FROM cypress/browsers:node8.9.3-chrome73
# Unzipping of Cypress binary very slow through npm install
# Instead, pull binary directly
# TODO: Verify checksum of binary
# See https://github.com/cypress-io/cypress/issues/812
RUN curl https://download.cypress.io/desktop/3.2.0?platform=linux64 -L -o cypress.zip
RUN mkdir -p /Cypress/3.2.0
RUN unzip -q cypress.zip -d /Cypress/3.2.0
ENV CYPRESS_CACHE_FOLDER=/Cypress

WORKDIR /repo
ADD web/app/cypress.json /repo/web/app/cypress.json
ADD web/app/cypress /repo/web/app/cypress
ADD Makefile /repo/Makefile
RUN CYPRESS_INSTALL_BINARY=0 CI=true npm i cypress@3.2.0
COPY --from=build-step /go/src/github.com/replicatedhq/ship/bin/ship /repo/bin/ship
CMD ["make", "cypress_base"]
