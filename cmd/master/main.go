package main

import (
	"github.com/DGHeroin/PluginSystem"
	"log"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	master := PluginSystem.NewMaster("127.0.0.1:7777")
	
	go func() {
		time.Sleep(time.Second)
		go master.StartPlugin(&PluginSystem.PluginInfo{
			Name:        "PingPlugin",
			Version:     "1.0.0",
			Conn:        nil,
			ExecutePath: "./build/plugins/plugin1",
		})
		
		go master.StartPlugin(&PluginSystem.PluginInfo{
			Name:        "PongPlugin",
			Version:     "1.0.0",
			Conn:        nil,
			ExecutePath: "./build/plugins/plugin2",
		})
	}()
	
	if err := master.Start(); err != nil {
		log.Fatalf("Master failed to start: %v", err)
	}
}
