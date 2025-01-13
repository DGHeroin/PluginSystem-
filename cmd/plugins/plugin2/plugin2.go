package main

import (
	"github.com/DGHeroin/PluginSystem/pkg/plugin"
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
	p.SetHandler("ping", func(ctx *plugin.Context) {
		data := ctx.GetRequestData()
		if string(data) != "ping" {
			ctx.ReplyData([]byte("unknown request"))
			return
		}
		ctx.ReplyData([]byte("pong"))
	})
	p.SetHandler("add", func(ctx *plugin.Context) {
		var req AddRequest
		if err := ctx.BindRequest(&req); err != nil {
			ctx.SetError(err)
			return
		}
		ctx.Reply(&AddResponse{
			Result: req.A + req.B,
		})
	})

	// 启动插件
	if err := p.Start(); err != nil {
		p.Logger().Error("Failed to start plugin", "error", err)
		return
	}
	select {}
}
