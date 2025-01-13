package main

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/DGHeroin/PluginSystem/pkg/plugin"
)

func main() {
	p := plugin.NewBasePlugin("PingPlugin", "1.0.0")

	// 启动插件
	if err := p.Start(); err != nil {
		p.Logger().Error("Failed to start plugin", "error", err)
		return
	}
	testAction := 1
	switch testAction {
	case 0:
		ticker := time.NewTicker(3 * time.Second)
		for range ticker.C {
			TestPing(p)
			TestAdd(p)
		}
	case 1:
		TestQPS(p)
	}
}

func TestAdd(p *plugin.BasePlugin) {
	if resp, err := p.SendRequest(context.Background(), "PongPlugin", "ping", []byte("ping")); err != nil {
		p.Logger().Error("Failed to send ping", "error", err)
	} else {
		p.Logger().Info("Ping", "result", resp)
	}
}

func TestPing(p *plugin.BasePlugin) {
	if resp, err := p.SendRequest(context.Background(), "PongPlugin", "add", []byte(`{"a": 1, "b": 1}`)); err != nil {
		p.Logger().Error("Failed to send ping", "error", err)
	} else {
		p.Logger().Info("Add", "result", resp)
	}
}

func TestQPS(p *plugin.BasePlugin) {
	const (
		testDuration   = 30 * time.Second // 测试持续时间
		reportInterval = time.Second      // 报告间隔
		concurrency    = 100              // 并发数
	)

	var (
		successCount int64
		failCount    int64
		totalLatency int64
	)

	// 创建上下文用于控制测试时间
	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	// 创建等待组用于等待所有goroutine完成
	var wg sync.WaitGroup

	// 启动统计goroutine
	ticker := time.NewTicker(reportInterval)
	defer ticker.Stop()

	go func() {
		lastSuccess := atomic.LoadInt64(&successCount)
		startTime := time.Now()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				currentSuccess := atomic.LoadInt64(&successCount)
				currentFails := atomic.LoadInt64(&failCount)
				currentLatency := atomic.LoadInt64(&totalLatency)

				qps := float64(currentSuccess-lastSuccess) / reportInterval.Seconds()
				avgLatency := float64(0)
				if currentSuccess > 0 {
					avgLatency = float64(currentLatency) / float64(currentSuccess) / float64(time.Millisecond)
				}

				p.Logger().Info("QPS Stats",
					"qps", qps,
					"total_requests", currentSuccess+currentFails,
					"success", currentSuccess,
					"fails", currentFails,
					"avg_latency_ms", avgLatency,
					"elapsed", time.Since(startTime).Seconds())

				lastSuccess = currentSuccess
			}
		}
	}()

	// 启动工作goroutine
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					start := time.Now()
					_, err := p.SendRequest(ctx, "PongPlugin", "add", []byte(`{"a": 1, "b": 1}`))
					latency := time.Since(start).Nanoseconds()

					if err != nil {
						atomic.AddInt64(&failCount, 1)
					} else {
						atomic.AddInt64(&successCount, 1)
						atomic.AddInt64(&totalLatency, latency)
					}
				}
			}
		}()
	}

	// 等待测试完成
	wg.Wait()

	// 输出最终统计结果
	finalSuccess := atomic.LoadInt64(&successCount)
	finalFails := atomic.LoadInt64(&failCount)
	finalLatency := atomic.LoadInt64(&totalLatency)
	totalRequests := finalSuccess + finalFails
	avgLatency := float64(0)
	if finalSuccess > 0 {
		avgLatency = float64(finalLatency) / float64(finalSuccess) / float64(time.Millisecond)
	}

	p.Logger().Info("Final QPS Test Results",
		"duration_sec", testDuration.Seconds(),
		"total_requests", totalRequests,
		"avg_qps", float64(totalRequests)/testDuration.Seconds(),
		"success_rate", float64(finalSuccess)/float64(totalRequests),
		"avg_latency_ms", avgLatency)
}
