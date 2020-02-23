package queue

import (
	"github.com/nsqio/go-nsq"
	"github.com/sirupsen/logrus"
	"github.com/wuzhc/gopusher/config"
	"github.com/wuzhc/gopusher/logger"
	"log"
	"os"
)

var Mq *Queue

type Queue struct {
	cfg      *config.Config
	Producer *Producer
	Consumer *Consumer
}

type Producer struct {
	obj *nsq.Producer
}

type Consumer struct {
	obj *nsq.Consumer
}

func NewQueue() *Queue {
	Mq = &Queue{}
	return Mq
}

func (q *Queue) InitProducer() error {
	nsqCfg := nsq.NewConfig()
	producer, err := nsq.NewProducer(config.Cfg.ProducerAddr, nsqCfg)
	if err != nil {
		return err
	}

	producer.SetLogger(log.New(os.Stdout, "", log.Flags()), nsq.LogLevelError)
	q.Producer = &Producer{
		obj: producer,
	}

	return nil
}

func (q *Queue) InitConsumer() error {
	nsqCfg := nsq.NewConfig()
	consumer, err := nsq.NewConsumer(config.Cfg.QueueTopic, config.Cfg.QueueChannel, nsqCfg)
	if err != nil {
		return err
	}

	consumer.SetLogger(log.New(os.Stdout, "", log.Flags()), nsq.LogLevelError)
	q.Consumer = &Consumer{
		obj: consumer,
	}

	return nil
}

func (q *Queue) Publish(msg []byte) error {
	return q.Producer.obj.Publish(config.Cfg.QueueTopic, msg)
}

func (q *Queue) Consume(fn nsq.HandlerFunc) {
	defer logger.Log().Println("queue consumer exit.")

	q.Consumer.obj.AddConcurrentHandlers(fn, config.Cfg.ConsumerHandlerNum)
	if err := q.Consumer.obj.ConnectToNSQD(config.Cfg.ConsumerAddr); err != nil {
		logger.Log().WithFields(logrus.Fields{"queue": "consumer"}).Fatalln(err)
		return
	}

	q.WaitConsume()
}

func (q *Queue) WaitConsume() {
	<-q.Consumer.obj.StopChan
}

func (q *Queue) StopConsume() {
	q.Consumer.obj.Stop()
}
