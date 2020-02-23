package socket

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/nsqio/go-nsq"
	"github.com/sirupsen/logrus"
	"github.com/wuzhc/gopusher/logger"
	pb "github.com/wuzhc/gopusher/proto"
	"github.com/wuzhc/gopusher/queue"
	"net/http"
	"sync"
	"time"
)

var Mg *Manager

type Handler func(client *Client, message interface{}) error

type Manager struct {
	wg            WaitGroupWrapper
	ctx           context.Context
	groups        sync.Map // map[string]*Group, group has many client
	clients       sync.Map // map[*Client]bool, all client connection
	registerCh    chan *Client
	unregisterCh  chan *Client
	handlers      map[string]Handler
	handlerMux    sync.RWMutex
	joinGroupMux  sync.Mutex
	unRegisterMux sync.Mutex
	exitChan      chan struct{}
}

func NewManager(ctx context.Context) (*Manager, error) {
	Mg = &Manager{
		ctx:          ctx,
		registerCh:   make(chan *Client),
		unregisterCh: make(chan *Client),
		handlers:     make(map[string]Handler),
		exitChan:     make(chan struct{}),
	}

	// Register default handler, don't overwrite them
	Mg.RegisterDefaultHandler()

	// Subscribe queue
	Mg.wg.Wrap(Mg.subscribeMq)

	// Manage clients
	Mg.wg.Wrap(Mg.manageClient)

	return Mg, nil
}

// the new value will overrides old value under the same name
func (m *Manager) RegisterHandler(name string, fn Handler) {
	m.handlerMux.Lock()
	defer m.handlerMux.Unlock()
	m.handlers[name] = fn
}

func (m *Manager) RegisterDefaultHandler() {
	m.RegisterHandler("join", Join)
	m.RegisterHandler("ping", Ping)
}

func (m *Manager) Exit() {
	logger.Log().Println("wait for close all group.")
	m.groups.Range(func(key, value interface{}) bool {
		value.(*Group).Close()
		return true
	})

	logger.Log().Println("wait for close all client.")
	m.clients.Range(func(key, value interface{}) bool {
		m.unregisterCh <- key.(*Client)
		return true
	})

	close(m.exitChan)
	m.wg.Wait()
	logger.Log().Println("manager exit.")
}

func (m *Manager) EstablishWS(ctx *gin.Context) {
	var upGrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	ws, err := upGrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}

	c := NewClient(ws, m)
	m.registerCh <- c
}

func (m *Manager) joinGroup(c *Client) error {
	defer func() {
		if err := recover(); err != nil {
			logger.Log().WithFields(logrus.Fields{"mananer": "joinGroup"}).Errorln(err)
		}
	}()

	if _, ok := m.clients.Load(c); !ok {
		return errors.New("client isn't establish ws")
	}

	value, _ := m.groups.LoadOrStore(c.CardID, NewGroup(c.CardID, m))
	value.(*Group).Join(c)
	return nil
}

// Push message to client from queue
func (m *Manager) subscribeMq() {
	queue.Mq.Consume(func(message *nsq.Message) error {
		defer message.Finish()

		var msg pb.PushRequest
		err := proto.Unmarshal(message.Body, &msg)
		if err != nil {
			return err
		}

		if len(msg.To) == 0 {
			return errors.New("no target to send.")
		}

		for _, receiver := range msg.To {
			v, ok := m.groups.Load(receiver)
			if !ok {
				continue
			}
			start := time.Now()
			v.(*Group).SendJson(msg)
			end := time.Now().Sub(start)
			logger.Log().WithFields(logrus.Fields{"time": end.Seconds(), "clientNum": len(v.(*Group).clients)}).Println("const time.")
		}

		logger.Log().WithFields(logrus.Fields{"queue": "subscribe"}).Println(msg.Content)
		return nil
	})
}

func (m *Manager) manageClient() {
	for {
		select {
		case <-m.exitChan:
			logger.Log().Println("exit manageClient")
			return
		case c := <-m.registerCh:
			m.clients.Store(c, true)
		case c := <-m.unregisterCh:
			if _, ok := m.clients.Load(c); !ok {
				continue
			}

			// Whether to remove group
			if v, ok := m.groups.Load(c.CardID); ok {
				group := v.(*Group)
				for k, client := range group.clients {
					if c != client { // Is it the current client
						continue
					}

					connNum := len(group.clients)

					// Remove client from group
					if connNum > 0 {
						if k == connNum-1 {
							group.clients = group.clients[:k]
						} else {
							group.clients = append(group.clients[:k], group.clients[k+1:]...)
						}
					}

					// It is the last client, remove group
					if len(group.clients) == 0 {
						group.Close()
						m.groups.Delete(c.CardID) // Remove group
						logger.Log().WithFields(logrus.Fields{"group": c.CardID}).Debugln("group is removed")
					}
					break
				}
			}

			logger.Log().WithFields(logrus.Fields{
				"conn":   c.Conn.RemoteAddr(),
				"cardID": c.CardID,
				"appID":  c.AppID,
			}).Debugln("client be removed.")

			c.Close()
			m.clients.Delete(c)
			c = nil
		}
	}
}

func (m *Manager) Stat() (int64, int64) {
	var groupNum int64
	var clientNum int64

	m.groups.Range(func(key, value interface{}) bool {
		groupNum++
		return true
	})
	m.clients.Range(func(key, value interface{}) bool {
		clientNum++
		return true
	})

	return clientNum, groupNum
}

func (m *Manager) IsOnline(cardID string) string {
	if _, ok := m.groups.Load(cardID); ok {
		return "yes"
	} else {
		return "no"
	}
}

func (m *Manager) GetGroupClientNum(cardID string) int64 {
	if v, ok := m.groups.Load(cardID); ok {
		return int64(len(v.(*Group).clients))
	} else {
		return -1
	}
}
