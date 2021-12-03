/*
Copyright 2021 IBM Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	gw "github.com/kserve/rest-proxy/gen"
)

const (
	restProxyPortEnvVar     = "REST_PROXY_LISTEN_PORT"
	restProxyGrpcPortEnvVar = "REST_PROXY_GRPC_PORT"
	restProxyTlsEnvVar      = "REST_PROXY_USE_TLS"
	tlsCertEnvVar           = "MM_TLS_KEY_CERT_PATH"
	tlsKeyEnvVar            = "MM_TLS_PRIVATE_KEY_PATH"
)

var (
	grpcServerEndpoint   = "localhost"
	inferenceServicePort = 8033
	logger               = zap.New()
	listenPort           = 8008
)

func run() error {
	logger.Info("Starting REST Proxy...")
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	marshaler := &CustomJSONPb{}
	marshaler.EmitUnpopulated = false
	marshaler.DiscardUnknown = false

	// Register gRPC server endpoint
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, marshaler),
	)

	var opts []grpc.DialOption
	if useTLS, ok := os.LookupEnv(restProxyTlsEnvVar); ok && useTLS == "true" {
		logger.Info("Using TLS")
		config := &tls.Config{
			InsecureSkipVerify: true,
		}
		opts = []grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(config)),
			grpc.WithBlock(),
		}
	} else {
		logger.Info("Not using TLS")
		opts = []grpc.DialOption{
			grpc.WithInsecure(),
			grpc.WithBlock(),
		}
	}

	if port, ok := os.LookupEnv(restProxyGrpcPortEnvVar); ok {
		grpcPort, err := strconv.Atoi(port)
		if err != nil {
			logger.Error(err, "unable to parse gRPC port environment variable")
			os.Exit(1)
		}
		inferenceServicePort = grpcPort
	}

	logger.Info("Registering gRPC Inference Service Handler", "Host", grpcServerEndpoint, "Port", inferenceServicePort)
	err := gw.RegisterGRPCInferenceServiceHandlerFromEndpoint(
		ctx, mux, fmt.Sprintf("%s:%d", grpcServerEndpoint, inferenceServicePort), opts)
	if err != nil {
		return err
	}

	if port, ok := os.LookupEnv(restProxyPortEnvVar); ok {
		listenPort, err = strconv.Atoi(port)
		if err != nil {
			logger.Error(err, "unable to parse port environment variable")
			os.Exit(1)
		}
	}

	// Start HTTP(S) server (and proxy calls to gRPC server endpoint)

	if certPath, ok := os.LookupEnv(tlsCertEnvVar); ok {
		keyPath := os.Getenv(tlsKeyEnvVar)
		logger.Info(fmt.Sprintf("Listening on port %d with TLS", listenPort))
		return http.ListenAndServeTLS(fmt.Sprintf(":%d", listenPort), certPath, keyPath, mux)
	}
	logger.Info(fmt.Sprintf("Listening on port %d", listenPort))
	return http.ListenAndServe(fmt.Sprintf(":%d", listenPort), mux)
}

func main() {
	if err := run(); err != nil {
		logger.Error(err, "unable to start gRPC REST proxy")
		os.Exit(1)
	}
}
