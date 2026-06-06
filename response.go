package ginx

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HTTP 状态码常量。
const (
	// CodeSuccess 表示请求成功。
	CodeSuccess = 0

	// CodeBadRequest 表示请求参数校验失败。
	CodeBadRequest = 400

	// CodeNotFound 表示请求的资源不存在。
	CodeNotFound = 404

	// CodeMethodNotAllowed 表示不支持的请求方法。
	CodeMethodNotAllowed = 405

	// CodeTooManyRequests 表示请求频率超限。
	CodeTooManyRequests = 429

	// CodeInternalError 表示服务器内部错误。
	CodeInternalError = 500

	// CodeServiceUnavailable 表示服务暂时不可用（如请求超时）。
	CodeServiceUnavailable = 503
)

// StandardizedResponse 是 ginx 统一的标准 JSON 响应体。
//
// 所有正常响应和异常兜底（404/405/429/500/503）均使用此结构。
// msg 字段使用简体中文。
type StandardizedResponse struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"requestId"`
	Timestamp int64  `json:"timestamp"`
}

// writeJSON 以标准化格式向客户端写入 JSON 响应并中止后续处理。
func writeJSON(c *gin.Context, httpStatus int, msg string, data any) {
	requestID := ""
	if id, exists := c.Get("requestId"); exists {
		if s, ok := id.(string); ok {
			requestID = s
		}
	}
	c.AbortWithStatusJSON(httpStatus, StandardizedResponse{
		Code:      httpStatus,
		Msg:       msg,
		Data:      data,
		RequestID: requestID,
		Timestamp: time.Now().UnixMilli(),
	})
}

// writeSuccessJSON 以标准化格式写入成功的 JSON 响应，使用 CodeSuccess (0) 作为 code。
func writeSuccessJSON(c *gin.Context, httpStatus int, msg string, data any) {
	requestID := ""
	if id, exists := c.Get("requestId"); exists {
		if s, ok := id.(string); ok {
			requestID = s
		}
	}
	c.AbortWithStatusJSON(httpStatus, StandardizedResponse{
		Code:      CodeSuccess,
		Msg:       msg,
		Data:      data,
		RequestID: requestID,
		Timestamp: time.Now().UnixMilli(),
	})
}

// noRouteHandler 返回 404 兜底 Handler，当请求路径无匹配时返回标准化 JSON。
func noRouteHandler(c *gin.Context) {
	writeJSON(c, http.StatusNotFound, "请求的资源不存在", nil)
}

// noMethodHandler 返回 405 兜底 Handler，当请求方法不匹配时返回标准化 JSON。
func noMethodHandler(c *gin.Context) {
	writeJSON(c, http.StatusMethodNotAllowed, "不支持的请求方法", nil)
}
