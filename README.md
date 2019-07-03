# qproxy [![Docker Repository on Quay](https://quay.io/repository/wish/qproxy/status "Docker Repository on Quay")](https://quay.io/repository/wish/qproxy)
qproxy is a single frontend API for multiple backend queueing systems. It supports gRPC as well as http/json via the gRPC-Gateway. For multi-cloud environments, or environments with multiple queueing systems, qproxy provides a unified interface and a single place for metrics, rate limiting, etc.

qproxy currently supports SQS as a backend, with plans to add GCP's PubSub.

# Building
You will need to follow [this](https://github.com/grpc-ecosystem/grpc-gateway#installation) to install protoc . Once that's done you should be able to run "make" to build the project.

# Running
```
./build/qproxy.linux --region=us-west-2
```
