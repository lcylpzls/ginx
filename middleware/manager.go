// Package middleware 提供 ginx 内置的 HTTP 中间件实现。
//
// 包含请求 ID 生成、跨域处理、请求超时、Panic 捕获、参数校验和 IP 限流等 6 个工业级中间件。
// 所有中间件通过 Manager 统一管理，支持禁用、覆盖和外部替换。
package middleware

import (
	"context"
	"sync"

	"github.com/gin-gonic/gin"
)

// Manager 管理内置中间件的注册表、执行顺序和启用状态。
//
// 调用方通过 Server 的方法间接操作 Manager，不直接使用此类型。
type Manager struct {
	builtins  map[string]gin.HandlerFunc
	overrides map[string]gin.HandlerFunc
	disabled  map[string]bool
	order     []string
	extras    []gin.HandlerFunc
	mu        sync.RWMutex
}

// NewManager 创建一个新的中间件管理器。
//
// 默认启用全部 6 个内置中间件（RateLimit 注册但默认禁用）。
func NewManager() *Manager {
	m := &Manager{
		builtins:  make(map[string]gin.HandlerFunc),
		overrides: make(map[string]gin.HandlerFunc),
		disabled:  make(map[string]bool),
		order: []string{
			string("recovery"),
			string("request_id"),
			string("timeout"),
			string("cors"),
			string("validation"),
			string("rate_limit"),
		},
		extras: make([]gin.HandlerFunc, 0),
	}

	// RateLimit 默认禁用
	m.disabled["rate_limit"] = true

	return m
}

// RegisterBuiltin 注册一个内置中间件到管理器。
//
// key 为中间件类型字符串，如 "request_id"、"cors" 等。
// handler 为中间件工厂函数返回的 gin.HandlerFunc。
func (m *Manager) RegisterBuiltin(key string, handler gin.HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.builtins[key] = handler
}

// Override 覆盖指定类型的内置中间件，使用调用方提供的自定义 Handler。
func (m *Manager) Override(mt string, handler gin.HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.overrides[mt] = handler
}

// Disable 禁用指定类型的内置中间件。
func (m *Manager) Disable(mt ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range mt {
		m.disabled[t] = true
	}
}

// Enable 启用指定类型的内置中间件。
//
// 注意：RateLimit 中间件必须通过 EnableRateLimit 激活，
// 仅调用 Enable 对其无效。
func (m *Manager) Enable(mt ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range mt {
		// RateLimit 不允许通过 Enable 激活
		if t == "rate_limit" {
			continue
		}
		delete(m.disabled, t)
	}
}

// EnableRateLimit 启用限流中间件并注册其 Handler。
func (m *Manager) EnableRateLimit(handler gin.HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.builtins["rate_limit"] = handler
	delete(m.disabled, "rate_limit")
}

// DisableRateLimit 禁用限流中间件并移除其 Handler。
func (m *Manager) DisableRateLimit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disabled["rate_limit"] = true
	delete(m.builtins, "rate_limit")
}

// Append 追加外部全局中间件到中间件链的末尾（路由专属中间件之前）。
func (m *Manager) Append(handler ...gin.HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.extras = append(m.extras, handler...)
}

// Build 构建最终执行的中间件链。
//
// 按照注册顺序返回已过滤（跳过禁用项、使用覆盖项）的 HandlerFunc 列表。
// 返回顺序：内置（启用）→ 外部全局 → 可用于路由专属。
func (m *Manager) Build(ctx context.Context) []gin.HandlerFunc {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var chain []gin.HandlerFunc

	for _, key := range m.order {
		// 跳过禁用的
		if m.disabled[key] {
			continue
		}

		// 优先使用覆盖
		if h, ok := m.overrides[key]; ok {
			chain = append(chain, h)
			continue
		}

		// 使用内置
		if h, ok := m.builtins[key]; ok {
			chain = append(chain, h)
		}
	}

	// 追加外部全局中间件
	chain = append(chain, m.extras...)

	return chain
}
