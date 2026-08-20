package repository

import (
	"errors"

	"forum-backend/database"
	"forum-backend/model"

	"gorm.io/gorm"
)

func ToggleLike(userID, postID uint) (bool, error) {
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
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		newLike := model.Like{
			UserID: userID,
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
	return isLiked, err
}

// 批量查某个用户对多个帖子有没有点过赞
func FindLikesByUserAndPosts(userID uint, postIDs []uint) ([]model.Like, error) {
	var likes []model.Like
	if err := database.DB.
		Where("user_id = ? AND post_id IN ?", userID, postIDs).
		Find(&likes).Error; err != nil {
		return nil, err
	}
	return likes, nil
}

// 查某个帖子下所有点赞记录，给 Redis 缓存用
func FindLikesByPostID(postID uint) ([]model.Like, error) {
	var likes []model.Like
	if err := database.DB.
		Select("user_id").
		Where("post_id = ?", postID).
		Find(&likes).Error; err != nil {
		return nil, err
	}
	return likes, nil
}
