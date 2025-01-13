package PluginSystem

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/DGHeroin/PluginSystem/pkg/protocol"
)

type PluginInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	// rw          io.ReadWriter
	conn        net.Conn
	ExecutePath string
	writeMu     sync.Mutex
}

type Master struct {
	address     string
	plugins     map[string]*PluginInfo
	pluginsLock sync.RWMutex
	logger      *slog.Logger
}

func NewMaster(address string) *Master {
	// 配置 slog
	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler).With("module", "Master")
	slog.SetDefault(logger)

	return &Master{
		address: address,
		plugins: make(map[string]*PluginInfo),
		logger:  logger,
	}
}

func (m *Master) Start() error {
	listener, err := net.Listen("tcp", m.address)
	if err != nil {
		return fmt.Errorf("failed to start listener: %v", err)
	}
	m.Logger().Info("Master listening on " + listener.Addr().String())

	for {
		conn, err := listener.Accept()
		if err != nil {
			m.Logger().Error("Accept error", "error", err)
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
		m.Logger().Error("Failed to decode register message", "error", err)
		conn.Close()
		return
	}

	// 注册插件并保存连接
	plugin := &PluginInfo{
		Name:    msg.Name,
		Version: msg.Version,
		conn:    conn,
	}

	m.pluginsLock.Lock()
	if oldPlugin, exists := m.plugins[msg.Name]; exists {
		// 如果已存在同名插件,关闭旧连接
		oldPlugin.conn.Close()
	}
	m.plugins[msg.Name] = plugin
	m.pluginsLock.Unlock()

	m.Logger().Info("Plugin registered", "name", msg.Name, "version", msg.Version)

	// 启动一个goroutine处理来自此插件的消息
	go m.handlePluginMessages(plugin, decoder)
}

func (m *Master) handlePluginMessages(plugin *PluginInfo, decoder *json.Decoder) {
	defer func() {
		plugin.conn.Close()
		m.pluginsLock.Lock()
		delete(m.plugins, plugin.Name)
		m.pluginsLock.Unlock()
		m.Logger().Info("Plugin disconnected", "name", plugin.Name)
	}()

	for {
		var msg protocol.Message
		if err := decoder.Decode(&msg); err != nil {
			m.Logger().Error("Failed to decode message from plugin", "plugin", plugin.Name, "error", err)
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
		m.Logger().Error("Target plugin not found", "plugin", msg.To)
		return
	}

	// 使用写锁保证消息发送的原子性
	targetPlugin.writeMu.Lock()
	encoder := json.NewEncoder(targetPlugin.conn)
	err := encoder.Encode(msg)
	targetPlugin.writeMu.Unlock()

	if err != nil {
		m.Logger().Error("Failed to forward message to plugin", "plugin", msg.To, "error", err)
	}
}
func (m *Master) StartPlugin(info ...*PluginInfo) error {
	for _, plugin := range info {
		if err := m.startPlugin(plugin); err != nil {
			return err
		}
	}
	return nil
}
func (m *Master) startPlugin(info *PluginInfo) error {
	execPath := info.ExecutePath
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(info.ExecutePath, ".exe") {
			execPath += ".exe"
		}
	}
	cmd := exec.Command(execPath)
	cmd.Env = append(os.Environ(), fmt.Sprintf("MASTER_ADDR=%s", m.address))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		m.Logger().Error("Failed to start plugin", "error", err)
	}
	return err
}
func (m *Master) Logger() *slog.Logger {
	return m.logger
}
