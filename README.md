# qproxy
qproxy is a single frontend API for multiple backend queueing systems. It supports gRPC as well as http/json via the gRPC-Gateway.

qproxy currently supports SQS as a backend, with plans to add GCP's PubSub.

# Building
You will need to follow [this](https://github.com/grpc-ecosystem/grpc-gateway#installation) to install protoc . Once that's done you should be able to run "make" to build the project.

# Running
```
./build/qproxy.linux --region=us-west-2
```
