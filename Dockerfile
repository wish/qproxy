FROM --platform=$BUILDPLATFORM golang:1.15 as builder

ARG BUILDPLATFORM
ARG TARGETARCH
ARG TARGETOS

RUN apt-get update && apt-get -y install git unzip
ADD https://github.com/google/protobuf/releases/download/v3.6.1/protoc-3.6.1-linux-x86_64.zip /
RUN unzip -o /protoc-3.6.1-linux-x86_64.zip -d /usr/local
COPY . /go/src/github.com/wish/qproxy
WORKDIR /go/src/github.com/wish/qproxy
ENV TARGETARCH=$TARGETARCH
RUN make build/qproxy.linux

FROM alpine:3.12
RUN apk --no-cache add ca-certificates
COPY --from=builder /go/src/github.com/wish/qproxy/build/qproxy.linux /bin/qproxy
CMD ["/bin/qproxy"]
