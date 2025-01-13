package protocol

// RegisterMessage 插件注册消息
type RegisterMessage struct {
	Name    string `json:"name"`    // 插件名称
	Version string `json:"version"` // 插件版本
}

// Message 插件间通信的消息
type Message struct {
	ID      int64  `json:"id"`              // 消息ID，用于关联请求和响应
	From    string `json:"from"`            // 发送者名称
	To      string `json:"to"`              // 接收者名称
	Type    string `json:"type"`            // 消息类型: request/response
	Method  string `json:"method"`          // 请求的方法名
	Payload []byte `json:"payload"`         // 消息内容
	Error   string `json:"error,omitempty"` // 错误信息，仅在响应时可能存在
}

// MessageType 定义消息类型常量
const (
	TypeRequest  = "request"
	TypeResponse = "response"
)
