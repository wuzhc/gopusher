package main

import (
	"context"
	"fmt"
	"github.com/etcd-io/etcd/clientv3"
	"github.com/wuzhc/gopusher/config"
	"github.com/wuzhc/gopusher/logger"
	"github.com/wuzhc/gopusher/queue"
	"github.com/wuzhc/gopusher/service"
	"github.com/wuzhc/gopusher/socket"
	"github.com/wuzhc/gopusher/web"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var wg sync.WaitGroup
var exitCh = make(chan struct{})

func main() {
	// Init config
	if err := config.InitConfig("config.ini"); err != nil {
		logger.Log().Fatalln(err)
	}

	// Init logger
	logger.InitLogger()

	// Init queue
	mq := queue.NewQueue()
	if err := mq.InitProducer(); err != nil {
		logger.Log().Fatalln(err)
	}
	if err := mq.InitConsumer(); err != nil {
		logger.Log().Fatalln(err)
	}

	// create manager for handing connection
	ctx, cancel := context.WithCancel(context.Background())
	manager, err := socket.NewManager(ctx)
	if err != nil {
		logger.Log().Fatalln(err)
	}

	// Register handler for websocket event
	manager.RegisterHandler("read", func(c *socket.Client, message interface{}) error {
		return nil
	})

	// Create gin web
	ginServer := web.NewGinServer()
	ginServer.UseMiddleware(web.Cors(), web.Logger())
	ginServer.RegisterRoute("/", web.Home)
	ginServer.RegisterRoute("/ws", manager.EstablishWS)
	ginServer.Start()

	// Grpc service
	grpcServer := service.NewGrpcServer(ctx)
	grpcServer.Start()
	grpcServer.StartGateway()

	// Register to etcd
	if config.Cfg.EtcdEnable {
		if err := registerToEtcd(); err != nil {
			logger.Log().Fatalln(err)
		}
	}

	// Register signal
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	logger.Log().Println("Gopusher is start.")
	<-ch

	cancel()
	close(exitCh)
	mq.StopConsume()
	grpcServer.Exit()
	ginServer.Exit()
	manager.Exit()

	wg.Wait()
	logger.Log().Println("done.")
}

func registerToEtcd() error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{config.Cfg.EtcdRegisteredAddr},
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := cli.Grant(ctx, 30)
	cancel()
	if err != nil {
		return err
	}

	listenKey := fmt.Sprintf("/gopusher/%s/%s", config.Cfg.EtcdListenKey, config.Cfg.GinServerAddr)
	logger.Log().Println(listenKey)
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	_, err = cli.Put(ctx, listenKey, config.Cfg.GinServerAddr+" weight=1", clientv3.WithLease(resp.ID))
	cancel()
	if err != nil {
		return err
	}

	respCh, err := cli.KeepAlive(context.TODO(), resp.ID)
	if err != nil {
		return err
	}

	go func() {
		wg.Add(1)
		for {
			select {
			case <-cli.Ctx().Done():
				goto exit
			case _, ok := <-respCh:
				if !ok {
					goto exit
				}
			case <-exitCh:
				goto exit
			}
		}
	exit:
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err := cli.Revoke(ctx, resp.ID)
		cancel()
		if err != nil {
			logger.Log().Errorln("etcd lease revoke failed, %s", err)
		}
		wg.Done()
		_ = cli.Close()
		logger.Log().Println("etcd lease keepalive exit.")
	}()

	return nil
}
