package ginx

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRoute_StructFields(t *testing.T) {
	handler := func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	}
	mw := func(c *gin.Context) {
		c.Next()
	}

	r := Route{
		Method:     "GET",
		Path:       "/api/users/:id",
		Handler:    handler,
		Middleware: []HandlerFunc{mw},
	}

	if r.Method != "GET" {
		t.Errorf("期望 Method 'GET'，实际 %q", r.Method)
	}
	if r.Path != "/api/users/:id" {
		t.Errorf("期望 Path '/api/users/:id'，实际 %q", r.Path)
	}
	if r.Handler == nil {
		t.Error("期望 Handler 非 nil")
	}
	if len(r.Middleware) != 1 {
		t.Errorf("期望 1 个 Middleware，实际 %d", len(r.Middleware))
	}
}

func TestRouteGroup_GET(t *testing.T) {
	handler := func(c *gin.Context) {}
	rg := &RouteGroup{prefix: "/api"}
	rg.GET("/users", handler)

	routes := rg.flatten()
	if len(routes) != 1 {
		t.Fatalf("期望 1 条路由，实际 %d", len(routes))
	}
	if routes[0].Method != "GET" {
		t.Errorf("期望 Method 'GET'，实际 %q", routes[0].Method)
	}
	if routes[0].Path != "/api/users" {
		t.Errorf("期望 Path '/api/users'，实际 %q", routes[0].Path)
	}
}

func TestRouteGroup_POST(t *testing.T) {
	handler := func(c *gin.Context) {}
	rg := &RouteGroup{prefix: "/api"}
	rg.POST("/users", handler)

	routes := rg.flatten()
	if len(routes) != 1 {
		t.Fatalf("期望 1 条路由，实际 %d", len(routes))
	}
	if routes[0].Method != "POST" {
		t.Errorf("期望 Method 'POST'，实际 %q", routes[0].Method)
	}
	if routes[0].Path != "/api/users" {
		t.Errorf("期望 Path '/api/users'，实际 %q", routes[0].Path)
	}
}

func TestRouteGroup_PUT(t *testing.T) {
	handler := func(c *gin.Context) {}
	rg := &RouteGroup{prefix: "/api"}
	rg.PUT("/users/:id", handler)

	routes := rg.flatten()
	if routes[0].Method != "PUT" {
		t.Errorf("期望 Method 'PUT'，实际 %q", routes[0].Method)
	}
}

func TestRouteGroup_DELETE(t *testing.T) {
	handler := func(c *gin.Context) {}
	rg := &RouteGroup{prefix: "/api"}
	rg.DELETE("/users/:id", handler)

	routes := rg.flatten()
	if routes[0].Method != "DELETE" {
		t.Errorf("期望 Method 'DELETE'，实际 %q", routes[0].Method)
	}
}

func TestRouteGroup_PATCH(t *testing.T) {
	handler := func(c *gin.Context) {}
	rg := &RouteGroup{prefix: "/api"}
	rg.PATCH("/users/:id", handler)

	routes := rg.flatten()
	if routes[0].Method != "PATCH" {
		t.Errorf("期望 Method 'PATCH'，实际 %q", routes[0].Method)
	}
}

func TestRouteGroup_Nested(t *testing.T) {
	handler := func(c *gin.Context) {}
	rg := &RouteGroup{prefix: "/api"}
	v2 := rg.Group("/v2")
	v2.GET("/users", handler)

	routes := rg.flatten()
	if len(routes) != 1 {
		t.Fatalf("期望 1 条路由，实际 %d", len(routes))
	}
	if routes[0].Path != "/api/v2/users" {
		t.Errorf("期望 Path '/api/v2/users'，实际 %q", routes[0].Path)
	}
}

func TestRouteGroup_MiddlewareInheritance(t *testing.T) {
	mw1 := func(c *gin.Context) { c.Next() }
	mw2 := func(c *gin.Context) { c.Next() }
	handler := func(c *gin.Context) {}

	rg := &RouteGroup{prefix: "/api"}
	rg.Use(mw1)
	rg.GET("/users", handler)

	// 子分组继承父中间件
	v2 := rg.Group("/v2")
	v2.Use(mw2)
	v2.GET("/products", handler)

	routes := rg.flatten()
	if len(routes) != 2 {
		t.Fatalf("期望 2 条路由，实际 %d", len(routes))
	}

	// 第一条：/api/users，仅有 mw1
	if len(routes[0].Middleware) != 1 {
		t.Errorf("/api/users 期望 1 个中间件，实际 %d", len(routes[0].Middleware))
	}

	// 第二条：/api/v2/products，有 mw1 + mw2
	if len(routes[1].Middleware) != 2 {
		t.Errorf("/api/v2/products 期望 2 个中间件，实际 %d", len(routes[1].Middleware))
	}
}

func TestRouteGroup_RouteSpecificMiddleware(t *testing.T) {
	mw1 := func(c *gin.Context) { c.Next() }
	mw2 := func(c *gin.Context) { c.Next() }
	mw3 := func(c *gin.Context) { c.Next() }
	handler := func(c *gin.Context) {}

	rg := &RouteGroup{prefix: "/api"}
	rg.Use(mw1)
	rg.GET("/users", handler, mw2, mw3)

	routes := rg.flatten()
	if len(routes) != 1 {
		t.Fatalf("期望 1 条路由，实际 %d", len(routes))
	}

	// 期望中间件顺序：mw1（分组级）, mw2, mw3（路由专属）
	if len(routes[0].Middleware) != 3 {
		t.Errorf("期望 3 个中间件，实际 %d", len(routes[0].Middleware))
	}
}

func TestRouteGroup_MultipleRoutes(t *testing.T) {
	handler := func(c *gin.Context) {}
	rg := &RouteGroup{prefix: "/api"}
	rg.GET("/users", handler)
	rg.POST("/users", handler)
	rg.GET("/products", handler)

	routes := rg.flatten()
	if len(routes) != 3 {
		t.Fatalf("期望 3 条路由，实际 %d", len(routes))
	}
}

func TestRouteGroup_EmptyGroup(t *testing.T) {
	rg := &RouteGroup{prefix: "/api"}
	routes := rg.flatten()
	if len(routes) != 0 {
		t.Errorf("期望 0 条路由，实际 %d", len(routes))
	}
}

func TestRouteGroup_DeepNested(t *testing.T) {
	handler := func(c *gin.Context) {}
	rg := &RouteGroup{prefix: "/api"}
	v1 := rg.Group("/v1")
	admin := v1.Group("/admin")
	admin.GET("/users", handler)

	routes := rg.flatten()
	if len(routes) != 1 {
		t.Fatalf("期望 1 条路由，实际 %d", len(routes))
	}
	if routes[0].Path != "/api/v1/admin/users" {
		t.Errorf("期望 Path '/api/v1/admin/users'，实际 %q", routes[0].Path)
	}
}

func TestRouteGroup_MiddlewareIsolation(t *testing.T) {
	// 验证不同分组中间件不互相污染
	mw1 := func(c *gin.Context) { c.Next() }
	mw2 := func(c *gin.Context) { c.Next() }
	handler := func(c *gin.Context) {}

	rg := &RouteGroup{prefix: "/api"}

	v1 := rg.Group("/v1")
	v1.Use(mw1)
	v1.GET("/users", handler)

	v2 := rg.Group("/v2")
	v2.Use(mw2)
	v2.GET("/users", handler)

	routes := rg.flatten()

	// v1 路由应只有 mw1
	v1Route := routes[0]
	if len(v1Route.Middleware) != 1 {
		t.Errorf("v1 路由期望 1 个中间件，实际 %d", len(v1Route.Middleware))
	}

	// v2 路由应只有 mw2
	v2Route := routes[1]
	if len(v2Route.Middleware) != 1 {
		t.Errorf("v2 路由期望 1 个中间件，实际 %d", len(v2Route.Middleware))
	}
}

func TestHandlerFunc_TypeAlias(t *testing.T) {
	// 验证 HandlerFunc 和 gin.HandlerFunc 完全等价
	var h1 HandlerFunc = func(c *gin.Context) {}
	var h2 gin.HandlerFunc = h1
	_ = h2
}
