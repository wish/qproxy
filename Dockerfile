FROM alpine:3.7
RUN apk add ca-certificates
ADD build/qproxy.linux /bin/qproxy
CMD ["/bin/qproxy"]
