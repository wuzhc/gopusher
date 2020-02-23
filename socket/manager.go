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
	wg           WaitGroupWrapper
	ctx          context.Context
	groups       map[string]*Group // group has many client
	clients      map[*Client]bool  // all client connection
	registerCh   chan *Client
	unregisterCh chan *Client
	handlers     map[string]Handler
	handlerMux   sync.RWMutex
	joinGroupMux sync.RWMutex
	exitChan     chan struct{}
}

func NewManager(ctx context.Context) (*Manager, error) {
	Mg = &Manager{
		ctx:          ctx,
		clients:      make(map[*Client]bool),
		registerCh:   make(chan *Client),
		unregisterCh: make(chan *Client),
		groups:       make(map[string]*Group),
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
	for _, g := range m.groups {
		g.Close()
	}

	logger.Log().Println("wait for close all client.")
	for c, _ := range m.clients {
		c.manager.unregisterCh <- c
	}

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
	var grp *Group

	m.joinGroupMux.Lock()

	if _, ok := m.clients[c]; !ok {
		return errors.New("client isn't establish ws")
	}
	if _, ok := m.groups[c.CardID]; ok {
		grp = m.groups[c.CardID]
	} else {
		grp = NewGroup(c.CardID, m)
		m.groups[c.CardID] = grp
	}

	m.joinGroupMux.Unlock()

	grp.Join(c)
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
			grp, ok := m.groups[receiver]
			if !ok {
				continue
			}
			start := time.Now()
			grp.SendJson(msg)
			end := time.Now().Sub(start)
			logger.Log().WithFields(logrus.Fields{"time": end.Seconds(), "clientNum": len(grp.clients)}).Println("const time.")
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
			m.clients[c] = true
		case c := <-m.unregisterCh:
			if _, ok := m.clients[c]; ok {
				// Whether to remove group
				group, ok := m.groups[c.CardID]
				if ok {
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
							delete(m.groups, c.CardID) // Remove group
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
				delete(m.clients, c)
				c = nil
			}
		}
	}
}

func (m *Manager) Stat() (int64, int64) {
	return int64(len(m.clients)), int64(len(m.groups))
}

func (m *Manager) IsOnline(cardID string) string {
	if group, ok := m.groups[cardID]; ok && len(group.clients) > 0 {
		return "yes"
	} else {
		return "no"
	}
}

func (m *Manager) GetGroupClientNum(cardID string) int64 {
	if group, ok := m.groups[cardID]; ok {
		return int64(len(group.clients))
	} else {
		return -1
	}
}
