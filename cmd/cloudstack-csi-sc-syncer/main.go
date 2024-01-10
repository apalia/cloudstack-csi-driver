// Small utility to synchronize CloudStack disk offerings to
// Kubernetes storage classes.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/apalia/cloudstack-csi-driver/pkg/syncer"
)

const agent = "cloudstack-csi-sc-syncer"

var (
	cloudstackconfig = flag.String("cloudstackconfig", "./cloud-config", "CloudStack configuration file")
	kubeconfig       = flag.String("kubeconfig", path.Join(os.Getenv("HOME"), ".kube/config"), "Kubernetes configuration file. Use \"-\" to use in-cluster configuration.")
	label            = flag.String("label", "app.kubernetes.io/managed-by="+agent, "")
	namePrefix       = flag.String("namePrefix", "cloudstack-", "")
	delete           = flag.Bool("delete", false, "Delete")
	volumeExpansion  = flag.Bool("volumeExpansion", false, "VolumeExpansion")
	showVersion      = flag.Bool("version", false, "Show version")

	// Version is set by the build process
	version = ""
)

func main() {
	flag.Parse()

	if *showVersion {
		baseName := path.Base(os.Args[0])
		fmt.Println(baseName, version)
		return
	}

	s, err := syncer.New(syncer.Config{
		Agent:            agent,
		CloudStackConfig: *cloudstackconfig,
		KubeConfig:       *kubeconfig,
		Label:            *label,
		NamePrefix:       *namePrefix,
		Delete:           *delete,
		VolumeExpansion:  *volumeExpansion,
	})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	err = s.Run(context.Background())
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	os.Exit(0)
}
