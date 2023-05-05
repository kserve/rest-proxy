[![Build](https://github.com/kserve/rest-proxy/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/kserve/rest-proxy/actions/workflows/build.yml)

# KServe V2 REST Proxy

This REST Proxy leverages [gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway) to create a reverse-proxy server which translates a RESTful HTTP API into gRPC. This allows sending inference requests using the [KServe V2 REST Predict Protocol](https://github.com/kserve/kserve/blob/master/docs/predict-api/v2/required_api.md#httprest) to platforms that expect the [gRPC V2 Predict Protocol](https://github.com/kserve/kserve/blob/master/docs/predict-api/v2/required_api.md#grpc).

**Note:** This is currently a work in progress, and is subject to performance and usability issues.

### Generate grpc-gateway stubs

```bash
protoc -I . --grpc-gateway_out ./gen/ --grpc-gateway_opt logtostderr=true --grpc-gateway_opt paths=source_relative grpc_predict_v2.proto
```

### Build Docker image

```bash
docker build -t kserve/rest-proxy:latest .
```
