module github.com/apalia/cloudstack-csi-driver

go 1.15

require (
	github.com/apache/cloudstack-go/v2 v2.9.1-0.20210727090705-0ad6453e08b8
	github.com/container-storage-interface/spec v1.5.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/hashicorp/go-uuid v1.0.2
	github.com/kubernetes-csi/csi-test/v4 v4.2.0
	go.uber.org/zap v1.16.0
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007
	golang.org/x/text v0.3.6
	google.golang.org/genproto v0.0.0-20210726200206-e7812ac95cc0 // indirect
	google.golang.org/grpc v1.39.0
	gopkg.in/gcfg.v1 v1.2.3
	gopkg.in/warnings.v0 v0.1.2 // indirect
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
	k8s.io/mount-utils v0.21.3
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
)
