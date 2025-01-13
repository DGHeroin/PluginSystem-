package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"

	"github.com/DGHeroin/PluginSystem/pkg/protocol"
)

type HandlerFunc func(*Context)
type Context struct {
	context.Context
	RequestReader  io.Reader
	ResponseWriter io.Writer
	err            error
}
type BasePlugin struct {
	Name       string
	Version    string
	Port       int
	conn       net.Conn
	r          *json.Decoder
	w          *json.Encoder
	handlers   map[string]HandlerFunc
	noRoute    HandlerFunc
	handlerMu  sync.RWMutex
	logger     *slog.Logger
	requestMap map[int64]*requestJob
	requestId  int64
	requestMu  sync.Mutex
}
type requestJob struct {
	request  *protocol.Message
	response *protocol.Message
	ch       chan *protocol.Message
	err      error
}

func NewBasePlugin(name, version string) *BasePlugin {
	// 配置 slog
	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler).With("plugin", name)
	slog.SetDefault(logger)

	return &BasePlugin{
		Name:       name,
		Version:    version,
		logger:     logger,
		handlers:   make(map[string]HandlerFunc),
		requestMap: make(map[int64]*requestJob),
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
		p.logger.Error("Failed to connect to master", "error", err)
		return fmt.Errorf("failed to connect to master: %v", err)
	}
	// 保存连接
	p.conn = conn
	p.r = json.NewDecoder(conn)
	p.w = json.NewEncoder(conn)

	// 发送注册消息
	if err := p.w.Encode(protocol.RegisterMessage{
		Name:    p.Name,
		Version: p.Version,
	}); err != nil {
		p.logger.Error("Failed to send register message", "error", err)
		return fmt.Errorf("failed to send register message: %v", err)
	}
	p.logger.Info("Plugin started")

	// 启动消息处理
	go p.handleConnection()

	return nil
}

func (p *BasePlugin) handleConnection() {
	defer p.conn.Close()

	responseCh := make(chan *protocol.Message)
	go func() {
		for msg := range responseCh {
			p.handleResponse(msg)
		}
	}()

	for {
		var msg protocol.Message
		if err := p.r.Decode(&msg); err != nil {
			p.logger.Error("Failed to decode message", "error", err)
			return
		}
		switch msg.Type {
		case protocol.TypeRequest:
			go p.handleRequest(&msg)
		case protocol.TypeResponse:
			responseCh <- &msg
		}
	}
}

func (p *BasePlugin) handleRequest(msg *protocol.Message) {
	response := &protocol.Message{
		ID:     msg.ID,
		From:   p.Name,
		To:     msg.From,
		Type:   protocol.TypeResponse,
		Method: msg.Method,
	}

	p.handlerMu.RLock()
	handler, exists := p.handlers[msg.Method]
	p.handlerMu.RUnlock()

	if !exists {
		if p.noRoute != nil {
			p.noRoute(&Context{
				RequestReader:  nil,
				ResponseWriter: nil,
			})
		} else {
			p.logger.Error("No handler for method", "method", msg.Method)
			response.Error = fmt.Sprintf("no handler for method: %s", msg.Method)
		}
	} else {
		ctx := &Context{
			RequestReader:  bytes.NewReader(msg.Payload),
			ResponseWriter: bytes.NewBuffer(nil),
		}
		handler(ctx)
		if ctx.err != nil {
			response.Error = ctx.err.Error()
		} else {
			response.Payload = ctx.ResponseWriter.(*bytes.Buffer).Bytes()
		}
	}

	if err := p.sendMessage(response); err != nil {
		p.logger.Error("Failed to send response", "error", err)
	}
}
func (p *BasePlugin) handleResponse(msg *protocol.Message) {
	p.requestMu.Lock()
	job, exists := p.requestMap[msg.ID]
	if exists {
		delete(p.requestMap, msg.ID)
	}
	p.requestMu.Unlock()
	if !exists {
		p.logger.Error("Received response for unknown request", "id", msg.ID)
		return
	}
	job.response = msg
	if msg.Error != "" {
		job.err = fmt.Errorf("response error: %s", msg.Error)
	}
	job.ch <- msg
}
func (p *BasePlugin) SetHandler(method string, handler HandlerFunc) {
	p.handlerMu.Lock()
	p.handlers[method] = handler
	p.handlerMu.Unlock()
}
func (p *BasePlugin) SetNoRouteHandler(handler HandlerFunc) {
	p.noRoute = handler
}

func (p *BasePlugin) sendMessage(msg *protocol.Message) error {
	if p.w == nil {
		return fmt.Errorf("not connected")
	}

	return p.w.Encode(msg)
}

func (p *BasePlugin) SendRequest(ctx context.Context, to string, method string, payload []byte) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	p.requestMu.Lock()
	p.requestId++
	request := &protocol.Message{
		ID:      p.requestId,
		From:    p.Name,
		To:      to,
		Type:    protocol.TypeRequest,
		Method:  method,
		Payload: payload,
	}
	job := &requestJob{
		request:  request,
		response: nil,
		ch:       make(chan *protocol.Message),
	}
	p.requestMap[p.requestId] = job
	p.requestMu.Unlock()

	p.sendMessage(request)

	// 等待响应
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-job.ch:
		var err = job.err

		if job.response.Error != "" {
			if err != nil {
				err = fmt.Errorf("%w: %s", err, job.response.Error)
			} else {
				err = fmt.Errorf("%s", job.response.Error)
			}
		}
		return job.response.Payload, err
	}
}

func (p *BasePlugin) Logger() *slog.Logger {
	return p.logger
}

func (c *Context) SetError(err error) {
	c.err = err
}
func (c *Context) Error() error {
	return c.err
}
func (c *Context) ReplyData(data []byte) {
	c.ResponseWriter.Write(data)
}
func (c *Context) Reply(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		c.SetError(err)
		return
	}
	c.ReplyData(data)
}
func (c *Context) GetRequestData() []byte {
	data, err := io.ReadAll(c.RequestReader)
	if err != nil {
		c.SetError(err)
		return nil
	}
	return data
}
func (c *Context) BindRequest(v interface{}) error {
	return json.NewDecoder(c.RequestReader).Decode(v)
}
