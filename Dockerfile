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
# Stage 1: Create the developer image for the BUILDPLATFORM only
###############################################################################
ARG BUILDPLATFORM="linux/amd64"
ARG GOLANG_VERSION=1.18.9
FROM --platform=${BUILDPLATFORM} registry.access.redhat.com/ubi8/go-toolset:${GOLANG_VERSION} AS develop

ARG PROTOC_VERSION=21.12

USER root

# Install build and dev tools
RUN true \
    && dnf install -y --nodocs \
    python3 \
       python3-pip \
       nodejs \
    && pip3 install pre-commit \
    && true

# https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Install protoc
# The protoc download files use a different variation of architecture identifiers
# from the Docker TARGETARCH forms amd64, arm64, ppc64le, s390x
#   protoc-22.2-linux-aarch_64.zip  <- arm64
#   protoc-22.2-linux-ppcle_64.zip  <- ppc64le
#   protoc-22.2-linux-s390_64.zip   <- s390x
#   protoc-22.2-linux-x86_64.zip    <- amd64
# so we need to map the arch identifiers before downloading the protoc.zip using
# shell parameter expansion: with the first character of a parameter being an
# exclamation point (!) it introduces a level of indirection where the value
# of the parameter is used as the name of another variable and the value of that
# other variable is the result of the expansion, e.g. the echo statement in the
# following three lines of shell script print "x86_64"
#   TARGETARCH=amd64
#   amd64=x86_64
#   echo ${!TARGETARCH}
RUN set -eux; \
    amd64=x86_64; \
    arm64=aarch_64; \
    ppc64le=ppcle_64; \
    s390x=s390_64; \
    wget -qO protoc.zip "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-${TARGETOS}-${!TARGETARCH}.zip" \
    && sha256sum protoc.zip \
    && unzip protoc.zip -x readme.txt -d /usr/local \
    && protoc --version \
    && true

WORKDIR /opt/app

COPY go.mod go.sum ./

# Install go protoc plugins
RUN go get google.golang.org/protobuf/cmd/protoc-gen-go \
           google.golang.org/grpc/cmd/protoc-gen-go-grpc

# Download and initialize the pre-commit environments before copying the source so they will be cached
COPY .pre-commit-config.yaml ./
RUN git init && \
    pre-commit install-hooks && \
    rm -rf .git

# Download dependiencies before copying the source so they will be cached
RUN go mod download

# the ubi/go-toolset image doesn't define ENTRYPOINT or CMD, but we need it to run 'make develop'
CMD /bin/bash


###############################################################################
# Stage 2: Run the go build with BUILDPLATFORM's native go compiler
###############################################################################
ARG BUILDPLATFORM="linux/amd64"
FROM --platform=${BUILDPLATFORM} develop AS build

LABEL image="build"

# Copy the source
COPY . ./

# https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
ARG TARGETOS=linux
ARG TARGETARCH=amd64

ARG GOOS=${TARGETOS}
ARG GOARCH=${TARGETARCH}

# Build the binaries using native go compiler from BUILDPLATFORM but compiled output for TARGETPLATFORM
# https://www.docker.com/blog/faster-multi-platform-builds-dockerfile-cross-compilation-guide/
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GO111MODULE=on go build -a -o /go/bin/server ./proxy/

###############################################################################
# Stage 3: Copy binaries only to create the smallest final runtime image
###############################################################################
FROM registry.access.redhat.com/ubi8/ubi-micro:8.7 as runtime

ARG USER=2000

USER ${USER}

COPY --from=build /go/bin/server /go/bin/server
CMD ["/go/bin/server"]
