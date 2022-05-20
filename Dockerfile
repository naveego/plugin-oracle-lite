# GO Version 1.13.15
FROM golang:1.13.15-stretch AS build-stage

# ENV Variables
ENV GO111MODULE=on

# Install compiilers
RUN apt-get update
RUN apt-get install mingw-w64 -y

# Install mage
WORKDIR /tmp
RUN wget -c https://github.com/magefile/mage/releases/download/v1.13.0/mage_1.13.0_Linux-ARM64.tar.gz
RUN tar -xzf mage_1.13.0_Linux-ARM64.tar.gz -C /go/bin/
RUN mage --version

# Load files to build
WORKDIR /build
COPY . ./

# build files
#RUN goreleaser --rm-dist
RUN mage Build

# Prepare Output
FROM scratch AS export-stage
WORKDIR /
COPY --from=build-stage /build/build .


