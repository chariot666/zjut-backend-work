package controller

import (
	"errors"
	"net/http"
	"strconv"

	"forum-backend/model"
	"forum-backend/service"
	"forum-backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreatePost(c *gin.Context) {
	var req struct {
		Content string `json:"content" binding:"required,min=1,max=2000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未认证")
		return
	}

	post, err := service.CreatePost(userID.(uint), req.Content)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "发布失败")
		return
	}
	utils.Created(c, post)
}

func GetPostList(c *gin.Context) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}

	items, total, err := service.ListPosts(page, pageSize, c.Query("sort"))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "获取帖子失败")
		return
	}

	utils.Success(c, gin.H{
		"items": items,
		"meta": gin.H{
			"page":      page,
			"page_size": pageSize,
			"total":     total,
		},
	})
}

func GetPostDetail(c *gin.Context) {
	postID, ok := parseIDParam(c, "post_id")
	if !ok {
		return
	}

	detail, err := service.GetPostDetail(postID)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "记录不存在")
		return
	}
	utils.Success(c, detail)
}

func DeletePost(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未认证")
		return
	}

	postID, ok := parseIDParam(c, "post_id")
	if !ok {
		return
	}

	if err := service.DeletePost(userID.(uint), postID); err != nil {
		writePostOperationError(c, err, "删除失败")
		return
	}
	utils.Success(c, nil)
}

func UpdatePost(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未认证")
		return
	}

	postID, ok := parseIDParam(c, "post_id")
	if !ok {
		return
	}

	var req struct {
		Content string `json:"content" binding:"required,min=1,max=2000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	post, err := service.UpdatePost(userID.(uint), postID, req.Content)
	if err != nil {
		writePostOperationError(c, err, "更新失败")
		return
	}
	utils.Success(c, post)
}

func AdminDeletePost(c *gin.Context) {
	postID, ok := parseIDParam(c, "post_id")
	if !ok {
		return
	}

	if err := service.AdminDeletePost(postID); err != nil {
		writePostOperationError(c, err, "删除失败")
		return
	}
	utils.Success(c, nil)
}

func parsePositiveInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func parseIDParam(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "参数错误")
		return 0, false
	}
	return uint(id), true
}

func writePostOperationError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		utils.Error(c, http.StatusNotFound, "记录不存在")
	case err.Error() == "禁止访问":
		utils.Error(c, http.StatusForbidden, "禁止访问")
	default:
		utils.Error(c, http.StatusInternalServerError, fallback)
	}
}

// Keep the legacy helper names available to the agent tools while the
// implementation lives in the service layer.
func buildPostResponse(post model.Post, commentCount int64) model.PostResponse {
	return service.BuildPostResponse(post, commentCount)
}

func countComments(postID uint) int64 {
	return service.CountComments(postID)
}

func postListOrder(sort string) string {
	return service.PostListOrder(sort)
}
