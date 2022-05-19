# GO Version 1.13.15
FROM golang:1.13.15-stretch

# Build Args
ARG GITHUB_TOKEN="token"

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

# Install goreleaser
WORKDIR /tmp
RUN wget -c https://github.com/goreleaser/goreleaser/releases/download/v1.9.0/goreleaser_Linux_x86_64.tar.gz
RUN tar -xzf goreleaser_Linux_x86_64.tar.gz -C /go/bin/
RUN goreleaser --version

# Load files to build
WORKDIR /build
COPY . ./

RUN ls -la

#RUN mage BuildWindows
#RUN ls build

# build files
RUN goreleaser --rm-dist



