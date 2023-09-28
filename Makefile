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

IMG_NAME ?= kserve/rest-proxy

# collect args from `make run` so that they don't run twice
ifeq (run,$(firstword $(MAKECMDGOALS)))
  RUN_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  ifneq ("$(wildcard /.dockerenv)","")
    $(error Inside docker container, run 'make $(RUN_ARGS)')
  endif
endif

.PHONY: all
## Alias for `generate build test`
all: generate build test

.PHONY: generate
## Generate GRPC gateway stubs
generate: google/api/annotations.proto google/api/http.proto
	protoc -I . --grpc-gateway_out ./gen/ --grpc-gateway_opt logtostderr=true --grpc-gateway_opt paths=source_relative grpc_predict_v2.proto

google/api/%.proto:
	@mkdir -p google/api
	@test -f $@ || wget --inet4-only -q -O $@ https://raw.githubusercontent.com/googleapis/googleapis/master/$@

.PHONY: build
## Build runtime Docker image
build:
	docker build -t ${IMG_NAME}:latest --target runtime .

.PHONY: build.develop
## Build develop container image
build.develop:
	docker build -t ${IMG_NAME}-develop:latest --target develop .

.PHONY: develop
## Run interactive shell inside developer container
develop: build.develop
	./scripts/develop.sh

.PHONY: run
## Run make target inside developer container (e.g. `make run fmt`)
run: build.develop
	./scripts/develop.sh make $(RUN_ARGS)

.PHONY: fmt
## Auto-format source code and report code-style violations (lint)
fmt:
	./scripts/fmt.sh

.PHONY: test
## Run tests
test:
	go test -coverprofile cover.out `go list ./...`

.DEFAULT_GOAL := help
.PHONY: help
## Print Makefile documentation
help:
	@perl -0 -nle 'printf("\033[36m  %-15s\033[0m %s\n", "$$2", "$$1") while m/^##\s*([^\r\n]+)\n^([\w.-]+):[^=]/gm' $(MAKEFILE_LIST) | sort

# Override targets if they are included in RUN_ARGs so it doesn't run them twice
# otherwise 'make run fmt' would be equivalent to calling './scripts/develop.sh make fmt'
# followed by 'make fmt'
$(eval $(RUN_ARGS):;@:)
