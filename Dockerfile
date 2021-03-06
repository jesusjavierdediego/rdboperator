FROM golang:alpine as golang
RUN apk add --no-cache git
RUN go get -v -tags=sqlite_vtable github.com/augmentable-dev/askgit
WORKDIR $GOPATH/src/xqledger/gitreader
COPY . ./
ADD https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 /usr/bin/dep
ADD resources/application.yml ./
COPY Gopkg.toml Gopkg.lock ./
RUN chmod +x /usr/bin/dep
RUN dep ensure --vendor-only
RUN CGO_ENABLED=0 go install -ldflags '-extldflags "-static"'
RUN apk --no-cache add tzdata zip ca-certificates
WORKDIR /usr/share/zoneinfo
RUN zip -r -0 /zoneinfo.zip .
ENV ZONEINFO /zoneinfo.zip
RUN apk add --update openssl && \
    rm -rf /var/cache/apk/*

COPY resources/application.yml /application.yml

ENTRYPOINT ["/go/bin/gitreader"]