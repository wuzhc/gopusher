package socket

import (
	"github.com/mitchellh/mapstructure"
)

// The event of join will bind the relationship between connection and user
// The user must be authenticated
// {"event":"join","data":{"card_id":"wuzhc","app_id":"ketang"}}
func Join(c *Client, message interface{}) error {
	var joinMsg JoinMsg
	if err := mapstructure.Decode(message, &joinMsg); err != nil {
		return err
	}

	c.CardID = joinMsg.CardID
	c.AppID = joinMsg.AppID
	if err := c.manager.joinGroup(c); err != nil {
		return err
	}

	return c.Conn.WriteJSON(ServerMsg{
		Type: 1,
		Data: "ok",
	})
}

// {"event":"ping"}}
func Ping(c *Client, message interface{}) error {
	return c.Conn.WriteJSON(ServerMsg{
		Type: 1,
		Data: "ok",
	})
}

