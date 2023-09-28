[![Build](https://github.com/kserve/rest-proxy/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/kserve/rest-proxy/actions/workflows/build.yml)

# KServe V2 REST Proxy

This REST Proxy leverages [gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway)
to create a reverse-proxy server which translates a RESTful HTTP API into gRPC.
This allows sending inference requests using the [KServe V2 REST Predict Protocol](https://github.com/kserve/kserve/blob/master/docs/predict-api/v2/required_api.md#httprest)
to platforms that expect the [gRPC V2 Predict Protocol](https://github.com/kserve/kserve/blob/master/docs/predict-api/v2/required_api.md#grpc).

**Note:** This is currently a work in progress, and is subject to performance and usability issues.

### Install the ProtoBuf compiler

The protocol buffer compiler, `protoc` is required to compile the `.proto` files.
To install it, follow the instructions [here](https://grpc.io/docs/protoc-installation/).

### Generate the gRPC gateway stubs

After changing the `grpc_predict_v2.proto` file, run the `protoc` compiler to regenerate
the gRPC gateway code stubs. It's recommended to use the developer image which has
all the required libraries pre-installed.

```bash
make run generate
```

### Build the Docker image

After regenerating the gRPC gateway stubs, rebuild the `rest-proxy` Docker image.

```bash
make build
```

### Push the Docker image

Before pushing the new `rest-proxy` image to your container registry, re-tag the
image created by `make build` in the step above.

```bash
DOCKER_USER="kserve"
DOCKER_TAG="dev"
docker tag kserve/rest-proxy:latest ${DOCKER_USER}/rest-proxy:${DOCKER_TAG}
docker push ${DOCKER_USER}/rest-proxy:${DOCKER_TAG}
```

### Update your ModelMesh deployment

In order to use the newly built `rest-proxy` image in a [ModelMesh Serving deployment](https://github.com/kserve/modelmesh-serving#modelmesh-serving) update the `restProxy.image` in [config/default/config-defaults.yaml](https://github.com/kserve/modelmesh-serving/blob/v0.11.0/config/default/config-defaults.yaml#L31-L32) and (re)deploy the ModelMesh Serving.

To update a running deployment of ModelMesh serving, add or update the `restProxy.image` section in the `model-serving-config` `ConfigMap` as described the [ModelMesh Serving configuration instructions](https://github.com/kserve/modelmesh-serving/tree/main/docs/configuration).
