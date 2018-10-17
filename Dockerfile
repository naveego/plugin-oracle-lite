FROM docker.n5o.black/dev/oracle-build

WORKDIR /src/github.com/naveego/plugin-oracle

RUN go get "github.com/naveego/ci/go/build"
RUN go get github.com/naveego/dataflow-contracts/plugins
RUN go get github.com/magefile/mage

RUN mage -v build

