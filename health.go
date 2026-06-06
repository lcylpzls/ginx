package ginx

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// healthData 健康检查响应中的 data 字段。
type healthData struct {
	Status  string `json:"status"`
	Uptime  string `json:"uptime"`
	Started string `json:"started"`
}

// healthHandler 返回一个健康检查 Handler。
//
// 返回标准化 JSON，包含服务状态、运行时间和启动时间。
// 请求路径通过 Config.HealthPath 自定义。
func healthHandler(startTime time.Time) HandlerFunc {
	return func(c *gin.Context) {
		uptime := time.Since(startTime)
		data := healthData{
			Status:  "运行中",
			Uptime:  formatUptime(uptime),
			Started: startTime.Format(time.RFC3339),
		}

		c.JSON(http.StatusOK, StandardizedResponse{
			Code:      CodeSuccess,
			Msg:       "ok",
			Data:      data,
			RequestID: c.GetString("requestId"),
			Timestamp: time.Now().UnixMilli(),
		})
	}
}

// formatUptime 将 time.Duration 格式化为人类可读的中文运行时间。
func formatUptime(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d秒", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		if s > 0 {
			return fmt.Sprintf("%d分钟%d秒", m, s)
		}
		return fmt.Sprintf("%d分钟", m)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if m > 0 {
		return fmt.Sprintf("%d小时%d分钟", h, m)
	}
	return fmt.Sprintf("%d小时", h)
}
