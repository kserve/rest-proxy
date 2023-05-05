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
all: build

.PHONY: build
build:
	docker build -t ${IMG_NAME}:latest --target runtime .

.PHONY: build.develop
build.develop:
	docker build -t ${IMG_NAME}-develop:latest --target develop .

.PHONY: develop
develop: build.develop
	./scripts/develop.sh

.PHONY: run
run: build.develop
	./scripts/develop.sh make $(RUN_ARGS)

.PHONY: fmt
fmt:
	./scripts/fmt.sh

.PHONY: test
test:
	go test -coverprofile cover.out `go list ./...`

# Override targets if they are included in RUN_ARGs so it doesn't run them twice
# otherwise 'make run fmt' would be equivalent to calling './scripts/develop.sh make fmt'
# followed by 'make fmt'
$(eval $(RUN_ARGS):;@:)
