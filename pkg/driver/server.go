package driver

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	//nolint:staticcheck // deprecated but still required by gRPC
	"github.com/golang/protobuf/proto"

	"github.com/container-storage-interface/spec/lib/go/csi"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"google.golang.org/grpc"
)

func (cs *cloudstackDriver) serve(ids csi.IdentityServer, ctrls csi.ControllerServer, ns csi.NodeServer) error {
	proto, addr, err := parseEndpoint(cs.endpoint)
	if err != nil {
		return err
	}

	if proto == "unix" {
		if !strings.HasPrefix(addr, "/") {
			addr = "/" + addr
		}
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("Failed to remove %s, error: %s", addr, err.Error())
		}
	}

	listener, err := net.Listen(proto, addr)
	if err != nil {
		return fmt.Errorf("Failed to listen: %w", err)
	}

	// Sanitize payload before logging
	grpc_zap.JsonPbMarshaller = sanitizer{}

	// Log every request and payloads (request + response)
	opts := []grpc.ServerOption{
		grpc_middleware.WithUnaryServerChain(
			grpc_zap.UnaryServerInterceptor(cs.logger),
			grpc_zap.PayloadUnaryServerInterceptor(cs.logger, func(context.Context, string, interface{}) bool { return true }),
		),
	}
	// Make sure that log statements internal to gRPC library are logged using the zapLogger as well.
	grpc_zap.ReplaceGrpcLoggerV2(cs.logger)

	server := grpc.NewServer(opts...)

	if ids != nil {
		csi.RegisterIdentityServer(server, ids)
	}
	if ctrls != nil {
		csi.RegisterControllerServer(server, ctrls)
	}
	if ns != nil {
		csi.RegisterNodeServer(server, ns)
	}

	cs.logger.Sugar().Infow("Listening for connections", "address", listener.Addr())
	return server.Serve(listener)
}

func parseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("Invalid endpoint: %v", ep)
}

type sanitizer struct{}

func (sanitizer) Marshal(out io.Writer, pb proto.Message) error {
	_, err := io.WriteString(out, protosanitizer.StripSecrets(pb).String())
	return err
}
