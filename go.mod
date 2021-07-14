module github.com/apalia/cloudstack-csi-driver

go 1.15

require (
	github.com/container-storage-interface/spec v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/hashicorp/go-uuid v1.0.2
	github.com/kubernetes-csi/csi-test/v4 v4.1.0
	github.com/xanzy/go-cloudstack/v2 v2.9.0
	go.uber.org/zap v1.16.0
	golang.org/x/text v0.3.5
	google.golang.org/grpc v1.36.0
	gopkg.in/gcfg.v1 v1.2.3
	gopkg.in/warnings.v0 v0.1.2 // indirect
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v0.20.4
	k8s.io/mount-utils v0.20.4
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
)
