package plugin

import (
	"encoding/json"
	"fmt"
	"github.com/DGHeroin/PluginSystem/pkg/protocol"
	"log"
	"net"
	"os"
)

type BasePlugin struct {
	Name    string
	Version string
	Port    int
	conn    net.Conn
	handler func(msg *protocol.Message) error
}

func NewBasePlugin(name, version string) *BasePlugin {
	return &BasePlugin{
		Name:    name,
		Version: version,
	}
}

func (p *BasePlugin) Start() error {
	// 获取主程序地址
	masterAddr := os.Getenv("MASTER_ADDR")
	if masterAddr == "" {
		return fmt.Errorf("MASTER_ADDR environment variable not set")
	}
	
	// 连接主程序
	conn, err := net.Dial("tcp", masterAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to master: %v", err)
	}
	p.conn = conn
	
	// 发送注册消息
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(protocol.RegisterMessage{
		Name:    p.Name,
		Version: p.Version,
	}); err != nil {
		return fmt.Errorf("failed to send register message: %v", err)
	}
	
	log.Printf("[Plugin] %s Plugin start ", p.Name)
	
	// 启动消息处理
	go p.handleConnection(conn)
	
	return nil
}

func (p *BasePlugin) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	decoder := json.NewDecoder(conn)
	for {
		var msg protocol.Message
		if err := decoder.Decode(&msg); err != nil {
			log.Printf("Failed to decode message: %v", err)
			return
		}
		
		if p.handler != nil {
			if err := p.handler(&msg); err != nil {
				log.Printf("Failed to handle message: %v", err)
			}
		}
	}
}

func (p *BasePlugin) SetHandler(handler func(msg *protocol.Message) error) {
	p.handler = handler
}

func (p *BasePlugin) SendMessage(to string, msgType string, payload []byte) error {
	msg := protocol.Message{
		From:    p.Name,
		To:      to,
		Type:    msgType,
		Payload: payload,
	}
	
	encoder := json.NewEncoder(p.conn)
	return encoder.Encode(msg)
}
