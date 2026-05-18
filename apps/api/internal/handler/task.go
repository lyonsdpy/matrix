package handler

import (
	"net/http"

	"matrix/api/internal/collector"
	"matrix/api/internal/model"
	"matrix/api/pkg/errno"

	"github.com/gin-gonic/gin"
)

// ListCollectors GET /api/v1/collectors
// 返回所有已注册采集器类型及描述，供创建任务时选择
func (h *Handler) ListCollectors(c *gin.Context) {
	c.JSON(http.StatusOK, collector.List())
}

// ListTasks GET /api/v1/tasks
func (h *Handler) ListTasks(c *gin.Context) {
	tasks, err := h.svc.Task.ListTasks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errno.New(500, "获取任务列表失败").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, tasks)
}

// GetTask GET /api/v1/tasks/:id
func (h *Handler) GetTask(c *gin.Context) {
	task, err := h.svc.Task.GetTask(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, errno.New(404, "任务不存在").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, task)
}

// CreateTask POST /api/v1/tasks
func (h *Handler) CreateTask(c *gin.Context) {
	var req model.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errno.New(400, "参数错误").WithErr(err))
		return
	}
	task, err := h.svc.Task.CreateTask(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errno.New(400, "创建任务失败").WithErr(err))
		return
	}
	c.JSON(http.StatusCreated, task)
}

// UpdateTask PUT /api/v1/tasks/:id
func (h *Handler) UpdateTask(c *gin.Context) {
	var req model.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errno.New(400, "参数错误").WithErr(err))
		return
	}
	task, err := h.svc.Task.UpdateTask(c.Param("id"), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, errno.New(400, "更新任务失败").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, task)
}

// DeleteTask DELETE /api/v1/tasks/:id
func (h *Handler) DeleteTask(c *gin.Context) {
	if err := h.svc.Task.DeleteTask(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, errno.New(404, "任务不存在").WithErr(err))
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// EnableTask POST /api/v1/tasks/:id/enable
func (h *Handler) EnableTask(c *gin.Context) {
	task, err := h.svc.Task.EnableTask(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, errno.New(404, "任务不存在").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, task)
}

// DisableTask POST /api/v1/tasks/:id/disable
func (h *Handler) DisableTask(c *gin.Context) {
	task, err := h.svc.Task.DisableTask(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, errno.New(404, "任务不存在").WithErr(err))
		return
	}
	c.JSON(http.StatusOK, task)
}

// RunTaskNow POST /api/v1/tasks/:id/run
// 立即触发一次采集，不等待下一个周期
func (h *Handler) RunTaskNow(c *gin.Context) {
	if err := h.svc.Task.RunNow(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, errno.New(404, "任务不存在").WithErr(err))
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "采集任务已触发"})
}
