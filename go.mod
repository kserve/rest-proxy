module github.com/kserve/rest-proxy

go 1.18

require (
	github.com/google/go-cmp v0.5.9
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.15.0
	google.golang.org/grpc v1.56.3
	google.golang.org/protobuf v1.30.0
	sigs.k8s.io/controller-runtime v0.14.1
)

require (
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/zapr v1.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apimachinery v0.26.0 // indirect
	k8s.io/klog/v2 v2.90.1 // indirect
	k8s.io/utils v0.0.0-20230209194617-a36077c30491 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)

replace (
	// fixes pkg/mod/github.com/golang/glog@v1.1.0/internal/logsink/logsink.go:123:41: undeclared name: any (requires version go1.18 or later)
	// remove when updating to go lang 1.19
	github.com/golang/glog => github.com/golang/glog v1.0.0
	golang.org/x/net => golang.org/x/net v0.17.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.27.0
)
