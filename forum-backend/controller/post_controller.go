package controller

import (
	"forum-backend/database"
	"forum-backend/model"
	"forum-backend/utils"
	"net/http"
	"strconv"

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
	post := model.Post{
		Content: req.Content,
	}
	post.UserID = userID.(uint)
	if err := database.DB.Create(&post).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, "发布失败")
		return
	}
	database.DB.Preload("User").First(&post, post.ID)
	utils.Created(c, buildPostResponse(post, 0))
}

func GetPostList(c *gin.Context) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}

	var total int64
	var posts []model.Post
	query := database.DB.Model(&model.Post{})
	if err := query.Count(&total).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, "获取帖子失败")
		return
	}

	order := postListOrder(c.Query("sort"))
	result := query.
		Preload("User").
		Order(order).
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&posts)
	if result.Error != nil {
		utils.Error(c, http.StatusInternalServerError, "获取帖子失败")
		return
	}

	items := make([]model.PostResponse, 0, len(posts))
	for _, post := range posts {
		items = append(items, buildPostResponse(post, countComments(post.ID)))
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
	id, ok := parseIDParam(c, "post_id")
	if !ok {
		return
	}
	var post model.Post
	result := database.DB.
		Preload("User").
		Preload("Comments", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at asc")
		}).
		Preload("Comments.User").
		First(&post, id)
	if result.Error != nil {
		utils.Error(c, http.StatusNotFound, "记录不存在")
		return
	}

	database.DB.Model(&model.Post{}).
		Where("id = ?", id).
		UpdateColumn(
			"view_count",
			gorm.Expr("view_count + ?", 1),
		)
	post.ViewCount++

	comments := make([]model.CommentResponse, 0, len(post.Comments))
	for _, comment := range post.Comments {
		comments = append(comments, comment.ToResponse())
	}
	utils.Success(c, model.PostDetailResponse{
		PostResponse: buildPostResponse(post, int64(len(comments))),
		Comments:     comments,
	})
}

func DeletePost(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未认证")
		return
	}
	id, ok := parseIDParam(c, "post_id")
	if !ok {
		return
	}
	var post model.Post
	result := database.DB.First(&post, id)
	if result.Error != nil {
		utils.Error(c, http.StatusNotFound, "记录不存在")
		return
	}
	if post.UserID != userID.(uint) {
		utils.Error(c, http.StatusForbidden, "禁止访问")
		return
	}
	if err := deletePostWithRelatedData(id); err != nil {
		utils.Error(c, http.StatusInternalServerError, "删除失败")
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
	id, ok := parseIDParam(c, "post_id")
	if !ok {
		return
	}
	var post model.Post
	result := database.DB.First(&post, id)
	if result.Error != nil {
		utils.Error(c, http.StatusNotFound, "记录不存在")
		return
	}
	if post.UserID != userID.(uint) {
		utils.Error(c, http.StatusForbidden, "禁止访问")
		return
	}
	var updateData struct {
		Content string `json:"content" binding:"required,min=1,max=2000"`
	}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	post.Content = updateData.Content
	if err := database.DB.Save(&post).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, "更新失败")
		return
	}
	database.DB.Preload("User").First(&post, id)
	utils.Success(c, buildPostResponse(post, countComments(post.ID)))
}

func AdminDeletePost(c *gin.Context) {
	id, ok := parseIDParam(c, "post_id")
	if !ok {
		return
	}
	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		utils.Error(c, http.StatusNotFound, "记录不存在")
		return
	}
	if err := deletePostWithRelatedData(id); err != nil {
		utils.Error(c, http.StatusInternalServerError, "删除失败")
		return
	}
	utils.Success(c, nil)
}

func buildPostResponse(post model.Post, commentCount int64) model.PostResponse {
	return model.PostResponse{
		ID:           post.ID,
		Content:      post.Content,
		Author:       post.User.ToResponse(),
		LikeCount:    cachedLikeCount(post),
		CommentCount: commentCount,
		ViewCount:    post.ViewCount,
		CreatedAt:    post.CreatedAt,
	}
}

func countComments(postID uint) int64 {
	var count int64
	database.DB.Model(&model.Comment{}).
		Where("post_id = ?", postID).
		Count(&count)
	return count
}

func postListOrder(sort string) string {
	if sort == "hot" {
		return "((like_count * 3) + (view_count * 1) + ((SELECT COUNT(*) FROM comments WHERE comments.post_id = posts.id) * 2)) desc, created_at desc"
	}
	return "created_at desc"
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

func deletePostWithRelatedData(postID uint) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("post_id = ?", postID).Delete(&model.Comment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&model.Like{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Post{}, postID).Error
	})
}
