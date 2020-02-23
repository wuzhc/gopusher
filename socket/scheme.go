package socket

// 客户端发来消息
type EventMsg struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data,omitempty"`
}

type JoinMsg struct {
	AppID  string `mapstructure:"app_id"`
	CardID string `mapstructure:"card_id"`
}

// redis推送的消息
type RedisMsg struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Content string   `json:"content"`
}

type ServerMsg struct {
	Type int    `json:"type"`
	Data string `json:"data"`
}

type Event struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type Bodycnt struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Content string `json:"content"`
}
