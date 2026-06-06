package ginx

import "github.com/gin-gonic/gin"

// HandlerFunc 是 gin.HandlerFunc 的类型别名。
//
// 这是 ginx 暴露的唯一 Gin 类型，调用方无需直接导入 Gin。
type HandlerFunc = gin.HandlerFunc

// Route 定义一条 HTTP 路由。
type Route struct {
	// Method HTTP 方法，如 GET、POST、PUT、DELETE、PATCH、HEAD、OPTIONS。
	Method string

	// Path 路由路径，如 "/api/users/:id"。
	Path string

	// Handler 路由处理器。
	Handler HandlerFunc

	// Middleware 路由专属中间件（可选），仅对当前路由生效。
	Middleware []HandlerFunc
}

// RouteGroup 路由分组，支持嵌套分组和分组级中间件。
//
// RouteGroup 仅缓冲注册，Start() 时一次性挂载到 Gin 引擎。
type RouteGroup struct {
	prefix     string
	routes     []Route
	groups     []*RouteGroup
	middleware []HandlerFunc
}

// GET 注册一条 GET 方法路由。
func (rg *RouteGroup) GET(path string, handler HandlerFunc, mw ...HandlerFunc) {
	rg.routes = append(rg.routes, Route{
		Method:     "GET",
		Path:       rg.prefix + path,
		Handler:    handler,
		Middleware: append(append([]HandlerFunc{}, rg.middleware...), mw...),
	})
}

// POST 注册一条 POST 方法路由。
func (rg *RouteGroup) POST(path string, handler HandlerFunc, mw ...HandlerFunc) {
	rg.routes = append(rg.routes, Route{
		Method:     "POST",
		Path:       rg.prefix + path,
		Handler:    handler,
		Middleware: append(append([]HandlerFunc{}, rg.middleware...), mw...),
	})
}

// PUT 注册一条 PUT 方法路由。
func (rg *RouteGroup) PUT(path string, handler HandlerFunc, mw ...HandlerFunc) {
	rg.routes = append(rg.routes, Route{
		Method:     "PUT",
		Path:       rg.prefix + path,
		Handler:    handler,
		Middleware: append(append([]HandlerFunc{}, rg.middleware...), mw...),
	})
}

// DELETE 注册一条 DELETE 方法路由。
func (rg *RouteGroup) DELETE(path string, handler HandlerFunc, mw ...HandlerFunc) {
	rg.routes = append(rg.routes, Route{
		Method:     "DELETE",
		Path:       rg.prefix + path,
		Handler:    handler,
		Middleware: append(append([]HandlerFunc{}, rg.middleware...), mw...),
	})
}

// PATCH 注册一条 PATCH 方法路由。
func (rg *RouteGroup) PATCH(path string, handler HandlerFunc, mw ...HandlerFunc) {
	rg.routes = append(rg.routes, Route{
		Method:     "PATCH",
		Path:       rg.prefix + path,
		Handler:    handler,
		Middleware: append(append([]HandlerFunc{}, rg.middleware...), mw...),
	})
}

// Use 向当前分组追加中间件，影响该分组内所有已注册和后续注册的路由。
func (rg *RouteGroup) Use(middleware ...HandlerFunc) {
	rg.middleware = append(rg.middleware, middleware...)
}

// Group 创建一个子分组，子分组继承父分组的 prefix 和中间件。
func (rg *RouteGroup) Group(relativePath string) *RouteGroup {
	subGroup := &RouteGroup{
		prefix:     rg.prefix + relativePath,
		middleware: append([]HandlerFunc{}, rg.middleware...),
	}
	rg.groups = append(rg.groups, subGroup)
	return subGroup
}

// flatten 递归展平分组中所有路由。
func (rg *RouteGroup) flatten() []Route {
	var all []Route
	all = append(all, rg.routes...)
	for _, g := range rg.groups {
		all = append(all, g.flatten()...)
	}
	return all
}
