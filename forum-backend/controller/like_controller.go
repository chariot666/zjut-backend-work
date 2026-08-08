package controller

import (
	"forum-backend/database"
	"forum-backend/model"
	"forum-backend/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 点赞；如果已经点赞，再次请求会取消点赞。
func LikePost(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未认证")
		return
	}
	postID, ok := parseIDParam(c, "post_id")
	if !ok {
		return
	}
	var post model.Post
	if err := database.DB.First(&post, postID).Error; err != nil {
		utils.Error(c, http.StatusNotFound, "记录不存在")
		return
	}

	isLiked := false
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var like model.Like
		err := tx.Where("user_id = ? AND post_id = ?", userID, postID).First(&like).Error
		if err == nil {
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			return tx.Model(&model.Post{}).
				Where("id = ? AND like_count > 0", postID).
				UpdateColumn("like_count", gorm.Expr("like_count - ?", 1)).
				Error
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		newLike := model.Like{
			UserID: userID.(uint),
			PostID: postID,
		}
		if err := tx.Create(&newLike).Error; err != nil {
			return err
		}
		isLiked = true
		return tx.Model(&model.Post{}).
			Where("id = ?", postID).
			UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).
			Error
	})
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "点赞失败")
		return
	}
	syncLikeCache(postID, userID.(uint), isLiked)
	utils.Success(
		c,
		gin.H{
			"post_id":  postID,
			"is_liked": isLiked,
		},
	)
}

// 旧接口保留为兼容入口，正式文档只使用 LikePost 的切换点赞。
func UnlikePost(c *gin.Context) {
	LikePost(c)
}

func GetLikeStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未认证")
		return
	}
	var req struct {
		PostIDs []uint `json:"post_ids" binding:"required,min=1,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	if database.RDB != nil {
		status, err := getLikeStatusFromRedis(userID.(uint), req.PostIDs)
		if err == nil {
			utils.Success(c, gin.H{"status": status})
			return
		}
	}

	var likes []model.Like
	if err := database.DB.
		Where("user_id = ? AND post_id IN ?", userID, req.PostIDs).
		Find(&likes).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, "查询失败")
		return
	}

	likedSet := make(map[uint]bool, len(likes))
	for _, like := range likes {
		likedSet[like.PostID] = true
	}
	status := make([]gin.H, 0, len(req.PostIDs))
	for _, postID := range req.PostIDs {
		status = append(status, gin.H{
			"post_id": postID,
			"liked":   likedSet[postID],
		})
	}
	utils.Success(c, gin.H{"status": status})
}

func syncLikeCache(postID uint, userID uint, isLiked bool) {
	if database.RDB == nil {
		return
	}

	var post model.Post
	if err := database.DB.Select("id", "like_count").First(&post, postID).Error; err != nil {
		return
	}

	userIDStr := strconv.FormatUint(uint64(userID), 10)
	likesKey := postLikesKey(postID)
	if isLiked {
		database.RDB.SRem(database.Ctx, likesKey, "__empty__")
		database.RDB.SAdd(database.Ctx, likesKey, userIDStr)
	} else {
		database.RDB.SRem(database.Ctx, likesKey, userIDStr)
		if post.LikeCount == 0 {
			database.RDB.SAdd(database.Ctx, likesKey, "__empty__")
		}
	}
	database.RDB.Set(database.Ctx, postLikeCountKey(postID), post.LikeCount, time.Hour)
	database.RDB.Expire(database.Ctx, likesKey, time.Hour)
}

func getLikeStatusFromRedis(userID uint, postIDs []uint) ([]gin.H, error) {
	status := make([]gin.H, 0, len(postIDs))
	userIDStr := strconv.FormatUint(uint64(userID), 10)

	for _, postID := range postIDs {
		if err := warmLikeCache(postID); err != nil {
			return nil, err
		}

		liked, err := database.RDB.SIsMember(database.Ctx, postLikesKey(postID), userIDStr).Result()
		if err != nil {
			return nil, err
		}
		status = append(status, gin.H{
			"post_id": postID,
			"liked":   liked,
		})
	}

	return status, nil
}

func warmLikeCache(postID uint) error {
	likesKey := postLikesKey(postID)
	countKey := postLikeCountKey(postID)

	exists, err := database.RDB.Exists(database.Ctx, likesKey, countKey).Result()
	if err != nil {
		return err
	}
	if exists == 2 {
		return nil
	}

	var post model.Post
	if err := database.DB.Select("id", "like_count").First(&post, postID).Error; err != nil {
		return err
	}

	var likes []model.Like
	if err := database.DB.Select("user_id").Where("post_id = ?", postID).Find(&likes).Error; err != nil {
		return err
	}

	if len(likes) > 0 {
		userIDs := make([]interface{}, 0, len(likes))
		for _, like := range likes {
			userIDs = append(userIDs, strconv.FormatUint(uint64(like.UserID), 10))
		}
		if err := database.RDB.SAdd(database.Ctx, likesKey, userIDs...).Err(); err != nil {
			return err
		}
	} else {
		if err := database.RDB.SAdd(database.Ctx, likesKey, "__empty__").Err(); err != nil {
			return err
		}
	}

	if err := database.RDB.Set(database.Ctx, countKey, post.LikeCount, time.Hour).Err(); err != nil {
		return err
	}
	if err := database.RDB.Expire(database.Ctx, likesKey, time.Hour).Err(); err != nil {
		return err
	}

	return nil
}

func cachedLikeCount(post model.Post) int {
	if database.RDB == nil {
		return post.LikeCount
	}

	value, err := database.RDB.Get(database.Ctx, postLikeCountKey(post.ID)).Int()
	if err == nil {
		return value
	}
	return post.LikeCount
}

func postLikesKey(postID uint) string {
	return "post:" + strconv.FormatUint(uint64(postID), 10) + ":likes"
}

func postLikeCountKey(postID uint) string {
	return "post:" + strconv.FormatUint(uint64(postID), 10) + ":like_count"
}
