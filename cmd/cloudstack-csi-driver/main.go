// cloudstack-csi-driver binary.
//
// To get usage information:
//
//    cloudstack-csi-driver -h
//
package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/apalia/cloudstack-csi-driver/pkg/cloud"
	"github.com/apalia/cloudstack-csi-driver/pkg/driver"
)

var (
	endpoint         = flag.String("endpoint", "unix:///tmp/csi.sock", "CSI endpoint")
	cloudstackconfig = flag.String("cloudstackconfig", "./cloudstack.ini", "CloudStack configuration file")
	nodeName         = flag.String("nodeName", "", "Node name")
	debug            = flag.Bool("debug", false, "Enable debug logging")
	showVersion      = flag.Bool("version", false, "Show version")

	// Version is set by the build process
	version  = ""
	isDevEnv = false
)

func main() {
	flag.Parse()

	if *showVersion {
		baseName := path.Base(os.Args[0])
		fmt.Println(baseName, version)
		return
	}

	if version == "" {
		isDevEnv = true
	}

	run()
	os.Exit(0)
}

func run() {
	// Setup logging
	var logConfig zap.Config
	if isDevEnv {
		logConfig = zap.NewDevelopmentConfig()
	} else {
		logConfig = zap.NewProductionConfig()
	}
	if *debug {
		logConfig.Level.SetLevel(zapcore.DebugLevel)
	}
	logger, _ := logConfig.Build()
	defer func() { _ = logger.Sync() }()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	// Setup cloud connector
	config, err := cloud.ReadConfig(*cloudstackconfig)
	if err != nil {
		logger.Sugar().Errorw("Cannot read CloudStack configuration", "error", err)
		os.Exit(1)
	}
	logger.Sugar().Debugf("Successfully read CloudStack configuration %v", *cloudstackconfig)
	csConnector := cloud.New(config)

	d, err := driver.New(*endpoint, csConnector, nil, *nodeName, version, logger)
	if err != nil {
		logger.Sugar().Errorw("Failed to initialize driver", "error", err)
		os.Exit(1)
	}

	if err = d.Run(); err != nil {
		logger.Sugar().Errorw("Server error", "error", err)
		os.Exit(1)
	}
}
