package main

import (
	"github.com/DGHeroin/PluginSystem"
	"log"
)

func main() {
	master := PluginSystem.NewMaster("127.0.0.1:7777")
	if err := master.Start(); err != nil {
		log.Fatalf("Master failed to start: %v", err)
	}
}
