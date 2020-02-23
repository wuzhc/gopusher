package socket

import (
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/wuzhc/gopusher/logger"
	"sync"
)

type Client struct {
	wg       WaitGroupWrapper
	Conn     *websocket.Conn
	AppID    string
	CardID   string
	manager  *Manager
	exitChan chan struct{}
	sync.Mutex
}

func NewClient(ws *websocket.Conn, manager *Manager) *Client {
	c := &Client{
		Conn:     ws,
		manager:  manager,
		exitChan: make(chan struct{}),
	}

	go c.listenEvent()
	return c
}

func (c *Client) Close() {
	close(c.exitChan)
	c.wg.Wait()
	_ = c.Conn.Close()
}

func (c *Client) listenEvent() {
	defer func() {
		c.manager.unregisterCh <- c
	}()

	var msg EventMsg
	var fields = logrus.Fields{
		"appID":  c.AppID,
		"cardID": c.CardID,
		"conn":   c.Conn.RemoteAddr().String(),
	}

	defer func() {
		if err := recover(); err != nil {
			logger.Log().WithFields(fields).Errorln(err)
		}
	}()

	for {
		select {
		case <-c.exitChan:
			logger.Log().Println("socket loop exit.")
			return
		default:
			if err := c.Conn.ReadJSON(&msg); err != nil {
				logger.Log().WithFields(fields).Debugln(err)
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte(err.Error()))
				return
			}

			// @link c.manger.RegisterHandler
			// @link eventhandler.go
			var err error
			if fn, ok := c.manager.handlers[msg.Event]; ok {
				err = fn(c, msg.Data)
			} else {
				err = NewFatalClientErr(ErrUnknownEvent, "unknown event")
			}

			if err != nil {
				logger.Log().WithFields(fields).Debugln(err)
				if _, ok := err.(*FatalClientErr); ok {
					_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte(err.Error()))
					return
				}
			}
		}
	}
}

func (c *Client) SendJson(msg string) error {
	c.Lock()
	defer c.Unlock()
	return c.Conn.WriteJSON(ServerMsg{
		Type: 1,
		Data: msg,
	})
}
