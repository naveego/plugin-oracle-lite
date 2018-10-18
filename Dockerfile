FROM golang:1.11-stretch


RUN apt-get update
RUN apt-get install mingw-w64 -y

RUN go get "github.com/naveego/ci/go/build"
RUN go get github.com/naveego/dataflow-contracts/plugins
RUN go get github.com/magefile/mage



