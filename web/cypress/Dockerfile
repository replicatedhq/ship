# Build Ship
FROM avcosystems/golang-node as build-step
ENV GOPATH=/go

RUN apt-get install bzip2

ADD . /go/src/github.com/replicatedhq/ship
WORKDIR /go/src/github.com/replicatedhq/ship/web
RUN CYPRESS_INSTALL_BINARY=0 yarn --frozen-lockfile
WORKDIR /go/src/github.com/replicatedhq/ship
RUN make build-ci

FROM cypress/browsers:chrome67
WORKDIR /repo
ADD web/cypress.json /repo/web/cypress.json
ADD web/cypress /repo/web/cypress
ADD Makefile /repo/Makefile
RUN npm i cypress
COPY --from=build-step /go/src/github.com/replicatedhq/ship/bin/ship /repo/bin/ship
CMD ["make", "cypress_base"]

# TODO: Unzipping of Cypress binary very slow, update to pull binary directly
# Example of pulling binary directly
# TODO: Figure out user permissions to make direct pull work
# RUN curl https://download.cypress.io/desktop/3.1.0?platform=linux64 -L -o cypress.zip
# RUN mkdir -p /Cypress/3.1.0
# RUN unzip -q cypress.zip -d /Cypress/3.1.0/Cypress
# RUN chmod +x /Cypress/3.1.0/Cypress/Cypress
# ENV CYPRESS_CACHE_FOLDER=/Cypress
# ...
# RUN CYPRESS_INSTALL_BINARY=0 npm i cypress
