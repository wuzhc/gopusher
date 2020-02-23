package web

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/wuzhc/gopusher/config"
	"net/http"
	"time"
)

type GinServer struct {
	engine *gin.Engine
	server *http.Server
}

func NewGinServer() *GinServer {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()
	return &GinServer{
		engine: engine,
	}
}

func (g *GinServer) UseMiddleware(middleware ...gin.HandlerFunc) {
	g.engine.Use(middleware...)
}

func (g *GinServer) RegisterRoute(uri string, fn gin.HandlerFunc) {
	g.engine.GET(uri, fn)
}

func (g *GinServer) Start() {
	g.server = &http.Server{Addr: config.Cfg.GinServerAddr, Handler: g.engine}
	go func() {
		_ = g.server.ListenAndServe()
	}()
}

func (serv *GinServer) Exit() {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	_ = serv.server.Shutdown(ctx)
}
