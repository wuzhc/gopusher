package socket

import (
	"github.com/sirupsen/logrus"
	"github.com/wuzhc/gopusher/config"
	"github.com/wuzhc/gopusher/logger"
	pb "github.com/wuzhc/gopusher/proto"
	"sync"
)

// All connection in group belong to the same cardID.
// Because one cardID can have multiple connections in multiple platforms.
type Group struct {
	name    string // cardID
	manager *Manager
	clients []*Client
	closed  bool
	sync.RWMutex
}

func NewGroup(name string, manager *Manager) *Group {
	return &Group{
		manager: manager,
		name:    name,
		clients: make([]*Client, 0),
	}
}

// Add client to group
func (g *Group) Join(client *Client) {
	fields := logrus.Fields{
		"group":      g.name,
		"appID":      client.AppID,
		"cardID":     client.CardID,
		"clientNum":  len(g.clients),
		"clientAddr": client.Conn.RemoteAddr(),
	}

	if g.closed {
		logger.Log().WithFields(fields).Warnln("group is closed.")
		return
	}

	g.RLock()
	for _, c := range g.clients {
		if c == client {
			logger.Log().WithFields(fields).Warnln("client don't join group again.")
			return
		}
	}
	g.RUnlock()

	g.Lock()
	g.clients = append(g.clients, client)
	g.Unlock()

	logger.Log().WithFields(fields).Debugln("join group successfully.")
}

// Send json message to each client in the group
func (g *Group) SendJson(msg pb.PushRequest) {
	var limit int
	var clientNum = len(g.clients)
	if config.Cfg.BatchSendNum > 0 {
		limit = clientNum / config.Cfg.BatchSendNum
	}

	if limit > 0 {
		var wg sync.WaitGroup
		startCh := make(chan struct{})
		for i := 0; i < limit; i++ {
			wg.Add(1)
			go func(i int) {
				<-startCh
				var end int
				start := i * config.Cfg.BatchSendNum
				if i == limit {
					end = clientNum - start
				} else {
					end = start + config.Cfg.BatchSendNum
				}
				g.doSend(start, end, msg)
				wg.Done()
			}(i)
		}
		close(startCh)
		wg.Wait()
	} else {
		g.doSend(0, clientNum, msg)
	}
}

func (g *Group) doSend(start, end int, msg pb.PushRequest) {
	g.RLock()
	defer g.RUnlock()

	for _, c := range g.clients[start:end] {
		fields := logrus.Fields{
			"group":      g.name,
			"message":    msg.Content,
			"connection": c.Conn.RemoteAddr(),
			"appID":      c.AppID,
			"cardID":     c.CardID,
		}

		// Skip other application connection
		if len(msg.AppID) > 0 && c.AppID != msg.AppID {
			continue
		}
		if err := c.SendJson(msg.Content); err != nil {
			g.manager.unregisterCh <- c
			logger.Log().WithFields(fields).Errorln(err)
			continue
		}
		logger.Log().WithFields(fields).Debugln("push success.")
	}
}

// close group
func (g *Group) Close() {
	g.RLock()
	defer g.RUnlock()

	g.closed = true
	for _, c := range g.clients {
		g.manager.unregisterCh <- c
	}
	g.clients = nil
}
