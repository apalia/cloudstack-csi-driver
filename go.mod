module github.com/apalia/cloudstack-csi-driver

go 1.15

require (
	github.com/container-storage-interface/spec v1.3.0
	github.com/golang/protobuf v1.4.3
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/hashicorp/go-uuid v1.0.2
	github.com/kubernetes-csi/csi-lib-utils v0.9.0
	github.com/kubernetes-csi/csi-test/v4 v4.0.2
	github.com/xanzy/go-cloudstack/v2 v2.9.0
	go.uber.org/zap v1.16.0
	golang.org/x/text v0.3.4
	google.golang.org/grpc v1.34.0
	gopkg.in/gcfg.v1 v1.2.3
	gopkg.in/warnings.v0 v0.1.2 // indirect
	k8s.io/api v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/client-go v0.20.1
	k8s.io/mount-utils v0.20.1
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
)
