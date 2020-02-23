package service

import (
	"github.com/golang/protobuf/proto"
	pb "github.com/wuzhc/gopusher/proto"
	"github.com/wuzhc/gopusher/queue"
	"github.com/wuzhc/gopusher/socket"
	"golang.org/x/net/context"
	"runtime"
)

type ApiService struct {
	context context.Context
}

// Push message
// curl -X POST -k http://127.0.0.1:8081/push -d '{"from":"sss","to":["wuzhc"], "content":"hellwo world"}'
func (service ApiService) Push(ctx context.Context, in *pb.PushRequest) (*pb.PushReply, error) {
	msg, err := proto.Marshal(in)
	if err != nil {
		return nil, err
	}

	if err := queue.Mq.Publish(msg); err != nil {
		return nil, err
	}

	in = nil
	msg = nil
	return &pb.PushReply{Code: 0, Message: "ok"}, nil
}

func (service ApiService) System(ctx context.Context, in *pb.SystemRequest) (*pb.SystemReply, error) {
	reply := &pb.SystemReply{
		CpuNum:       int64(runtime.NumCPU()),
		GoroutineNum: int64(runtime.NumGoroutine()),
	}
	reply.ClientNum, reply.GroupNum = socket.Mg.Stat()
	return reply, nil
}

func (service ApiService) IsOnline(ctx context.Context, in *pb.IsOnlineRequest) (*pb.IsOnlineReply, error) {
	return &pb.IsOnlineReply{
		Result: socket.Mg.IsOnline(in.CardID),
	}, nil
}

func (service ApiService) Group(ctx context.Context, in *pb.GroupRequest) (*pb.GroupReply, error) {
	return &pb.GroupReply{
		ClientNum: socket.Mg.GetGroupClientNum(in.CardID),
	}, nil
}
