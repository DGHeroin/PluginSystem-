package main

import (
	"encoding/json"
	"github.com/DGHeroin/PluginSystem/pkg/plugin"
	"github.com/DGHeroin/PluginSystem/pkg/protocol"
	"log"
)

type AddRequest struct {
	A int `json:"a"`
	B int `json:"b"`
}

type AddResponse struct {
	Result int `json:"result"`
}

func main() {
	p := plugin.NewBasePlugin("PongPlugin", "1.0.0")
	
	// 处理ping和加法请求
	p.SetHandler(func(msg *protocol.Message) error {
		log.Printf("[PongPlugin] Received message type: %s", msg.Type)
		switch msg.Type {
		case "ping":
			// 收到ping就回pong
			log.Printf("Received ping from %s", msg.From)
			return p.SendMessage(msg.From, "pong", []byte("pong"))
		
		case "add":
			// 处理加法请求
			var req AddRequest
			if err := json.Unmarshal(msg.Payload, &req); err != nil {
				log.Printf("Failed to unmarshal add request: %v", err)
				return err
			}
			
			// 计算结果
			resp := AddResponse{
				Result: req.A + req.B,
			}
			
			// 编码并发送响应
			payload, err := json.Marshal(resp)
			if err != nil {
				return err
			}
			
			log.Printf("Calculated %d + %d = %d", req.A, req.B, resp.Result)
			return p.SendMessage(msg.From, "add_response", payload)
		}
		return nil
	})
	
	// 启动插件
	if err := p.Start(); err != nil {
		log.Fatalf("Failed to start plugin: %v", err)
	}
	
}
