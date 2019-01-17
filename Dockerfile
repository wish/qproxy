FROM golang:1.11-stretch
RUN apt-get update && apt-get -y install git unzip
ENV GOPATH /go
ENV PATH ${GOPATH}/bin:$PATH
RUN go get -u github.com/golang/dep/cmd/dep
ADD https://github.com/google/protobuf/releases/download/v3.6.1/protoc-3.6.1-linux-x86_64.zip /
RUN unzip -o /protoc-3.6.1-linux-x86_64.zip -d /usr/local bin/protoc
RUN unzip -o /protoc-3.6.1-linux-x86_64.zip -d /usr/local
RUN git clone https://github.com/wish/qproxy.git /go/src/github.com/wish/qproxy
WORKDIR /go/src/github.com/wish/qproxy
RUN make build/qproxy.linux

FROM alpine:3.7
RUN apk --no-cache add ca-certificates
COPY --from=0 /go/src/github.com/wish/qproxy/build/qproxy.linux /bin/qproxy
CMD ["/bin/qproxy"]
