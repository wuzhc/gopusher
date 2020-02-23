package service

import (
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/sirupsen/logrus"
	"github.com/wuzhc/gopusher/config"
	"github.com/wuzhc/gopusher/logger"
	pb "github.com/wuzhc/gopusher/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"os"
)

type GrpcServer struct {
	ctx      context.Context
	inner    *grpc.Server
	exitChan chan struct{}
}

func NewGrpcServer(ctx context.Context) *GrpcServer {
	return &GrpcServer{
		ctx:      ctx,
		inner:    grpc.NewServer(),
		exitChan: make(chan struct{}),
	}
}

func (server *GrpcServer) Start() {
	listener, err := net.Listen("tcp", config.Cfg.GrpcServerAddr)
	if err != nil {
		logger.Log().WithFields(logrus.Fields{"grpcServerAddr": config.Cfg.GrpcServerAddr}).Errorln(err)
		os.Exit(1) // exit process
	}

	go func() {
		select {
		case <-server.exitChan:
			if err := listener.Close(); err != nil {
				logger.Log().WithFields(logrus.Fields{"grpcServer": "listener.Close"}).Errorln(err)
			}
		}
	}()

	pb.RegisterRpcServer(server.inner, ApiService{
		context: server.ctx,
	})
	go func() {
		if err := server.inner.Serve(listener); err != nil {
			logger.Log().WithFields(logrus.Fields{"grpcServer": "server.Serve"}).Debugln(err)
		}
	}()
}

func (server *GrpcServer) StartGateway() {
	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := pb.RegisterRpcHandlerFromEndpoint(server.ctx, mux, config.Cfg.GrpcServerAddr, opts)
	if err != nil {
		logger.Log().Errorln(err)
		os.Exit(-1)
	}

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	go func() {
		defer logger.Log().Println("grpc gateway exit.")
		if err := http.ListenAndServe(config.Cfg.GrpcGatewayAddr, mux); err != nil {
			logger.Log().Errorln(err)
		}
	}()
}

func (server *GrpcServer) Exit() {
	close(server.exitChan)
}
