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

all: build

build:
	docker build -t ${IMG_NAME}:latest --target runtime .

build.develop:
	docker build -t ${IMG_NAME}-develop:latest --target develop .

fmt:
	./scripts/fmt.sh

test:
	go test -coverprofile cover.out `go list ./...`
