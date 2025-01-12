package PluginSystem

import (
	"encoding/json"
	"fmt"
	"github.com/DGHeroin/PluginSystem/pkg/protocol"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
)

type PluginInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Conn        net.Conn
	ExecutePath string
	writeMu     sync.Mutex
}

type Master struct {
	address     string
	plugins     map[string]*PluginInfo
	pluginsLock sync.RWMutex
}

func NewMaster(address string) *Master {
	return &Master{
		address: address,
		plugins: make(map[string]*PluginInfo),
	}
}

func (m *Master) Start() error {
	listener, err := net.Listen("tcp", m.address)
	if err != nil {
		return fmt.Errorf("failed to start listener: %v", err)
	}
	log.Printf("Master listening on %s", listener.Addr())
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go m.handleConnection(conn)
	}
}

func (m *Master) handleConnection(conn net.Conn) {
	// 读取注册消息
	var msg protocol.RegisterMessage
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&msg); err != nil {
		log.Printf("Failed to decode register message: %v", err)
		conn.Close()
		return
	}
	
	// 注册插件并保存连接
	plugin := &PluginInfo{
		Name:    msg.Name,
		Version: msg.Version,
		Conn:    conn,
	}
	
	m.pluginsLock.Lock()
	if oldPlugin, exists := m.plugins[msg.Name]; exists {
		// 如果已存在同名插件,关闭旧连接
		oldPlugin.Conn.Close()
	}
	m.plugins[msg.Name] = plugin
	m.pluginsLock.Unlock()
	
	log.Printf("Plugin registered: %s (version: %s)", msg.Name, msg.Version)
	
	// 启动一个goroutine处理来自此插件的消息
	go m.handlePluginMessages(plugin, decoder)
}

func (m *Master) handlePluginMessages(plugin *PluginInfo, decoder *json.Decoder) {
	defer func() {
		plugin.Conn.Close()
		m.pluginsLock.Lock()
		delete(m.plugins, plugin.Name)
		m.pluginsLock.Unlock()
		log.Printf("Plugin disconnected: %s", plugin.Name)
	}()
	
	for {
		var msg protocol.Message
		if err := decoder.Decode(&msg); err != nil {
			log.Printf("Failed to decode message from %s: %v", plugin.Name, err)
			return
		}
		
		// 处理或转发消息
		m.forwardMessage(&msg)
	}
}

func (m *Master) forwardMessage(msg *protocol.Message) {
	m.pluginsLock.RLock()
	targetPlugin, exists := m.plugins[msg.To]
	m.pluginsLock.RUnlock()
	
	if !exists {
		log.Printf("Target plugin not found: %s", msg.To)
		return
	}
	
	// 使用写锁保证消息发送的原子性
	targetPlugin.writeMu.Lock()
	encoder := json.NewEncoder(targetPlugin.Conn)
	err := encoder.Encode(msg)
	targetPlugin.writeMu.Unlock()
	
	if err != nil {
		log.Printf("Failed to forward message to %s: %v", msg.To, err)
	}
}

func (m *Master) StartPlugin(info *PluginInfo) error {
	cmd := exec.Command(info.ExecutePath)
	cmd.Env = append(os.Environ(), fmt.Sprintf("MASTER_ADDR=%s", m.address))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}
