package handler

import (
	"net/http"
	"strconv"

	"matrix/api/pkg/errno"

	"github.com/gin-gonic/gin"
)

// StartEndpointSync POST /api/v1/endpoints/sync/start
// 触发一次飞书设备同步。同一时刻只允许一个任务，重复调用返回当前 job_id（started=false）。
func (h *Handler) StartEndpointSync(c *gin.Context) {
	jobID, started, err := h.svc.EndpointSync.Start(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, errno.New(503, "设备同步暂不可用").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"job_id": jobID, "started": started})
}

// GetEndpointSyncProgress GET /api/v1/endpoints/sync/progress
// 查询设备同步进度快照。前端按 ~500ms 间隔轮询，phase=done/failed 时停止。
func (h *Handler) GetEndpointSyncProgress(c *gin.Context) {
	c.JSON(http.StatusOK, h.svc.EndpointSync.Snapshot())
}

// ListEndpoints GET /api/v1/endpoints
// 终端管理-列表参数：
//   ?q=设备名/序列号模糊
//   ?user=关联用户(当前 OR 最近)姓名/邮箱筛选
//   ?type=物理形态精确匹配（PC|LAPTOP|PRINTER|TV|ATTENDANCE|PHONE|TABLET|OTHER）
//   ?os=操作系统精确匹配（WINDOWS|MACOS|LINUX|IOS|ANDROID|HARMONYOS|OTHER）
//   ?cursor=游标 ?limit=页大小
func (h *Handler) ListEndpoints(c *gin.Context) {
	q := c.Query("q")
	userQ := c.Query("user")
	typeFilter := c.Query("type")
	osFilter := c.Query("os")
	cursor := c.Query("cursor")
	limit, _ := strconv.Atoi(c.Query("limit"))

	list, err := h.svc.Endpoint.List(c.Request.Context(), q, userQ, typeFilter, osFilter, cursor, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errno.New(500, "查询终端失败").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, list)
}

// GetEndpoint GET /api/v1/endpoints/:id
// 终端管理-详情：:id 为节点 id（SHA1(feishu_device_id) 派生）。
func (h *Handler) GetEndpoint(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, errno.New(400, "missing endpoint id"))
		return
	}
	ep, err := h.svc.Endpoint.GetDetail(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errno.New(500, "查询终端失败").WithErr(err))
		return
	}
	if ep == nil {
		c.JSON(http.StatusNotFound, errno.New(404, "终端不存在"))
		return
	}
	c.JSON(http.StatusOK, ep)
}
