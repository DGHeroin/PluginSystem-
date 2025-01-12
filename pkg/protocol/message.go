package protocol

// RegisterMessage 插件注册消息
type RegisterMessage struct {
	Name    string `json:"name"`    // 插件名称
	Version string `json:"version"` // 插件版本
}

// Message 插件间通信的消息
type Message struct {
	From    string `json:"from"`    // 发送者名称
	To      string `json:"to"`      // 接收者名称
	Type    string `json:"type"`    // 消息类型
	Payload []byte `json:"payload"` // 消息内容
}
