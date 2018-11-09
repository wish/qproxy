FROM alpine:3.7
ADD build/qproxy.linux /bin/qproxy
CMD ["/bin/qproxy"]
