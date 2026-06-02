package handler

import (
	"net/http"
	"strconv"
	"strings"

	"matrix/api/pkg/errno"

	"github.com/gin-gonic/gin"
)

// ListDepartmentChildren GET /api/v1/contacts/tree?parent=<id>
// 通讯录-部门树懒加载：返回某部门的直接子部门。parent 为空或 "0" 时返回顶级部门。
// 每个子节点带 has_children 标志，便于前端判断是否还能展开。
func (h *Handler) ListDepartmentChildren(c *gin.Context) {
	parent := c.Query("parent")
	nodes, err := h.svc.Department.ListChildren(c.Request.Context(), parent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errno.New(500, "查询部门树失败").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"departments": nodes})
}

// SearchContacts GET /api/v1/contacts/search?q=<keyword>&limit=<n>
// 通讯录-联合搜索：一次输入同时检索用户和部门，结果按类型分组返回。
func (h *Handler) SearchContacts(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	res, err := h.svc.Contact.Search(c.Request.Context(), q, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errno.New(500, "搜索失败").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, res)
}

// StartContactSync POST /api/v1/contacts/sync/start
// 触发一次飞书通讯录同步。同一时刻只允许一个任务，重复调用返回当前 job_id（started=false）。
func (h *Handler) StartContactSync(c *gin.Context) {
	jobID, started, err := h.svc.Sync.Start(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, errno.New(503, "同步暂不可用").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"job_id": jobID, "started": started})
}

// GetContactSyncProgress GET /api/v1/contacts/sync/progress
// 查询同步进度快照。前端按 ~500ms 间隔轮询，phase=done/failed 时停止。
func (h *Handler) GetContactSyncProgress(c *gin.Context) {
	c.JSON(http.StatusOK, h.svc.Sync.Snapshot())
}
