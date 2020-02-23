package service

import (
	"context"
	"github.com/gogo/protobuf/proto"
	"github.com/nsqio/go-nsq"
	"github.com/wuzhc/gopusher/config"
	"github.com/wuzhc/gopusher/logger"
	pb "github.com/wuzhc/gopusher/proto"
	"github.com/wuzhc/gopusher/queue"
	"google.golang.org/grpc"
	"runtime"
	"sync"
	"testing"
	"time"
)

// go test ./... -v
func TestApiService_Push(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.GrpcServerAddr = "127.0.0.1:9004"
	cfg.LogLevel = 2
	ctx, cancel := context.WithCancel(context.Background())

	// Create queue
	mq := queue.NewQueue(cfg)
	if err := mq.InitProducer(); err != nil {
		t.Fatal(err)
	}
	if err := mq.InitConsumer(); err != nil {
		t.Fatal(err)
	}

	// Init logger
	logger.InitLogger(cfg)

	// Create grpc
	grpcServer := NewGrpcServer(cfg, ctx)
	grpcServer.Start()
	time.Sleep(3 * time.Second) // enable server start

	conn, err := grpc.Dial("127.0.0.1:9004", grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Contact the server and print out its response.
	c := pb.NewRpcClient(conn)
	r, err := c.Push(context.Background(), &pb.PushRequest{
		From:    "xxxx",
		To:      []string{"xxx", "wuzhc"},
		Content: "hello world",
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.Message == "ok" {
		t.Logf("pass, result:%s", r.Message)
	}

	startCh := make(chan struct{})
	var wg sync.WaitGroup

	go func() {
		wg.Add(1)
		<-startCh
		queue.Mq.Consume(func(message *nsq.Message) error {
			defer message.Finish()
			var msg pb.PushRequest
			err := proto.Unmarshal(message.Body, &msg)
			if err != nil {
				t.Error(err)
			}
			if msg.Content == "hello world" {
				t.Logf("pass, result:%s", msg.Content)
			}
			wg.Done()
			return nil
		})
	}()

	close(startCh)
	wg.Wait()
	queue.Mq.StopConsume()

	cancel()
	grpcServer.Exit()
	time.Sleep(1 * time.Second)
}

// BenchmarkApiService_Push-4   	   20000	     94412 ns/op
func BenchmarkApiService_Push(b *testing.B) {
	b.StopTimer()
	cfg := config.NewDefaultConfig()
	cfg.GrpcServerAddr = "127.0.0.1:9004"
	cfg.LogLevel = 2
	ctx, cancel := context.WithCancel(context.Background())

	// Init Producer for queue
	mq := queue.NewQueue(cfg)
	if err := mq.InitProducer(); err != nil {
		b.Fatal(err)
	}

	// Init logger
	logger.InitLogger(cfg)

	// Start Grpc Server
	grpcServer := NewGrpcServer(cfg, ctx)
	grpcServer.Start()

	conn, err := grpc.Dial("127.0.0.1:9004", grpc.WithInsecure())
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	c := pb.NewRpcClient(conn)
	startCh := make(chan struct{})
	var wg sync.WaitGroup
	parallel := runtime.GOMAXPROCS(0)

	req := &pb.PushRequest{
		From:    "wuzhc",
		To:      []string{"wuzhc"},
		Content: "hello world",
	}
	for j := 0; j < parallel; j++ {
		wg.Add(1)
		go func() {
			<-startCh
			for i := 0; i < b.N/parallel; i++ {
				_, _ = c.Push(context.Background(), req)
			}
			wg.Done()
		}()
	}

	b.StartTimer()
	close(startCh)
	wg.Wait()
	cancel()
	grpcServer.Exit()
}
