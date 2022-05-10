# Copyright 2021 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

###############################################################################
# Stage 1: Create the develop, test, and build environment
###############################################################################
FROM  registry.access.redhat.com/ubi8/ubi-minimal:8.4 AS develop

ARG GOLANG_VERSION=1.16.6
ARG PROTOC_VERSION=3.14.0

USER root

# Install build and dev tools
RUN microdnf install \
    gcc \
    gcc-c++ \
    make \
    vim \
    findutils \
    diffutils \
    git \
    wget \
    tar \
    unzip \
    python3 \
    nodejs && \
    pip3 install pre-commit

# Install go
ENV PATH /usr/local/go/bin:$PATH
RUN set -eux; \
    wget -qO go.tgz "https://golang.org/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz"; \
    sha256sum *go.tgz; \
    tar -C /usr/local -xzf go.tgz; \
    go version

# Install protoc
RUN set -eux; \
    wget -qO protoc.zip "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip"; \
    sha256sum protoc.zip; \
    unzip protoc.zip -x readme.txt -d /usr/local; \
    protoc --version

# Install go protoc plugins
ENV PATH /root/go/bin:$PATH
RUN go get google.golang.org/protobuf/cmd/protoc-gen-go \
           google.golang.org/grpc/cmd/protoc-gen-go-grpc

WORKDIR /opt/app

# Download and initialize the pre-commit environments before copying the source so they will be cached
COPY .pre-commit-config.yaml ./
RUN git init && \
    pre-commit install-hooks && \
    rm -rf .git

# Download dependiencies before copying the source so they will be cached
COPY go.mod go.sum ./
RUN go mod download

###############################################################################
# Stage 2: Run the build
###############################################################################
FROM develop AS build

LABEL image="build"

# Copy the source
COPY . ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o /go/bin/server ./proxy/

###############################################################################
# Stage 3: Copy binary to create the smallest final runtime image
###############################################################################
FROM scratch AS runtime

ARG USER=2000

USER ${USER}

COPY --from=build /go/bin/server /go/bin/server
CMD ["/go/bin/server"]
