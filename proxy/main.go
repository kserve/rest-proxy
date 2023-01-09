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
	"google.golang.org/grpc/credentials/insecure"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	gw "github.com/kserve/rest-proxy/gen"
)

const (
	restProxyPortEnvVar     = "REST_PROXY_LISTEN_PORT"
	restProxyGrpcMaxMsgSize = "REST_PROXY_GRPC_MAX_MSG_SIZE_BYTES"
	restProxyGrpcPortEnvVar = "REST_PROXY_GRPC_PORT"
	restProxyTlsEnvVar      = "REST_PROXY_USE_TLS"
	tlsCertEnvVar           = "MM_TLS_KEY_CERT_PATH"
	tlsKeyEnvVar            = "MM_TLS_PRIVATE_KEY_PATH"
)

var (
	grpcServerEndpoint = "localhost"
	logger             = zap.New()

	// Defaults
	inferenceServicePort    = 8033
	listenPort              = 8008
	maxGrpcMessageSizeBytes = 16777216
)

func getIntegerEnv(envVar string, defaultValue int) int {
	if val, ok := os.LookupEnv(envVar); ok {
		val, err := strconv.Atoi(val)
		if err != nil {
			logger.Error(err, "unable to parse environment variable", "env", envVar)
			os.Exit(1)
		}
		return val
	}
	return defaultValue
}

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

	maxGrpcMessageSizeBytes = getIntegerEnv(restProxyGrpcMaxMsgSize, maxGrpcMessageSizeBytes)

	var opts []grpc.DialOption
	var transportCreds credentials.TransportCredentials
	if useTLS, ok := os.LookupEnv(restProxyTlsEnvVar); ok && useTLS == "true" {
		logger.Info("Using TLS")
		transportCreds = credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})
	} else {
		logger.Info("Not using TLS")
		transportCreds = insecure.NewCredentials()
	}
	opts = []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxGrpcMessageSizeBytes)),
	}
	inferenceServicePort = getIntegerEnv(restProxyGrpcPortEnvVar, inferenceServicePort)

	logger.Info("Registering gRPC Inference Service Handler", "Host", grpcServerEndpoint, "Port", inferenceServicePort, "MaxCallRecvMsgSize", maxGrpcMessageSizeBytes)
	err := gw.RegisterGRPCInferenceServiceHandlerFromEndpoint(
		ctx, mux, fmt.Sprintf("%s:%d", grpcServerEndpoint, inferenceServicePort), opts)
	if err != nil {
		return err
	}

	listenPort = getIntegerEnv(restProxyPortEnvVar, listenPort)

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
