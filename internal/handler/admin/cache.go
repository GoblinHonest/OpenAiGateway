package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/example/aigateway/internal/cache"
	"github.com/gin-gonic/gin"
)

// CacheHandler 缓存管理Handler
type CacheHandler struct {
	cache *cache.Cache
}

func NewCacheHandler(c *cache.Cache) *CacheHandler {
	return &CacheHandler{cache: c}
}

// GetConfig 获取缓存配置
func (h *CacheHandler) GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"enabled":     h.cache.Enabled(),
		"ttl":         h.cache.TTL().String(),
		"type":        "llm_response_cache",
		"description": "网关侧 LLM 响应缓存，相同请求命中时直接返回缓存结果",
	})
}

// UpdateConfig 更新缓存配置（运行时生效，仅支持 enabled 和 ttl）
func (h *CacheHandler) UpdateConfig(c *gin.Context) {
	var req struct {
		Enabled *bool   `json:"enabled"`
		TTL     *string `json:"ttl"` // e.g. "5m", "1h"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Enabled != nil {
		h.cache.SetEnabled(*req.Enabled)
	}
	if req.TTL != nil {
		dur, err := time.ParseDuration(*req.TTL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ttl format: " + err.Error()})
			return
		}
		h.cache.SetTTL(dur)
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled": h.cache.Enabled(),
		"ttl":     h.cache.TTL().String(),
		"message": "缓存配置已更新（运行时生效，重启后恢复配置文件值）",
	})
}

// GetStats 获取缓存统计
func (h *CacheHandler) GetStats(c *gin.Context) {
	count, err := h.cache.Stats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled":      h.cache.Enabled(),
		"total_entries": count,
	})
}

// Clear 清除所有缓存
func (h *CacheHandler) Clear(c *gin.Context) {
	if err := h.cache.Clear(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "缓存已清除",
	})
}

// ListEntries 列出缓存条目（分页返回实际条目）
func (h *CacheHandler) ListEntries(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	entries, total, err := h.cache.ListEntries(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     entries,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
