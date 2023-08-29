FROM ubuntu:16.04

ARG GO_VERSION
ENV GO_VERSION=1.21.0

RUN apt-get update
RUN apt-get install -y wget git gcc build-essential libx11-dev libxcursor-dev libglfw3-dev pkg-config

RUN wget -P /tmp "https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz"

RUN tar -C /usr/local -xzf "/tmp/go${GO_VERSION}.linux-amd64.tar.gz"
RUN rm "/tmp/go${GO_VERSION}.linux-amd64.tar.gz"

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

WORKDIR /app

CMD go build .