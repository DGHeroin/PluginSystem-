package main

import (
	"github.com/DGHeroin/PluginSystem/pkg/plugin"
	"github.com/DGHeroin/PluginSystem/pkg/protocol"
	"log"
	"time"
)

func main() {
	p := plugin.NewBasePlugin("PingPlugin", "1.0.0")
	
	// 处理pong响应
	p.SetHandler(func(msg *protocol.Message) error {
		if msg.Type == "pong" {
			log.Printf("Received pong from %s", msg.From)
		}
		return nil
	})
	
	// 启动插件
	if err := p.Start(); err != nil {
		log.Fatalf("Failed to start plugin: %v", err)
	}
	
	// 每3秒发送一次ping
	ticker := time.NewTicker(3 * time.Second)
	for range ticker.C {
		if err := p.SendMessage("PongPlugin", "ping", []byte("ping")); err != nil {
			log.Printf("Failed to send ping: %v", err)
		} else {
			log.Printf("Sent ping to PongPlugin")
		}
	}
}
