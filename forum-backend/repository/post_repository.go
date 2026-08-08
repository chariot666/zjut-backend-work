package repository

import (
	"errors"

	"forum-backend/database"
	"forum-backend/model"

	"gorm.io/gorm"
)

func CreatePost(post *model.Post) error {
	return database.DB.Create(post).Error
}

func FindPostByID(postID uint) (*model.Post, error) {
	var post model.Post
	if err := database.DB.Preload("User").First(&post, postID).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func FindPostDetailByID(postID uint) (*model.Post, error) {
	var post model.Post
	if err := database.DB.
		Preload("User").
		Preload("Comments", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at asc")
		}).
		Preload("Comments.User").
		First(&post, postID).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func ListPosts(order string, limit, offset int) ([]model.Post, error) {
	var posts []model.Post
	if err := database.DB.
		Model(&model.Post{}).
		Preload("User").
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func CountPosts() (int64, error) {
	var total int64
	if err := database.DB.Model(&model.Post{}).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func CountComments(postID uint) (int64, error) {
	var count int64
	if err := database.DB.Model(&model.Comment{}).
		Where("post_id = ?", postID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func IncrementViewCount(postID uint) error {
	return database.DB.Model(&model.Post{}).
		Where("id = ?", postID).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).
		Error
}

func DeletePostWithRelatedData(postID uint) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("post_id = ?", postID).Delete(&model.Comment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&model.Like{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.Post{}, postID).Error; err != nil {
			return err
		}
		return nil
	})
}

func UpdatePostContent(postID uint, content string) error {
	return database.DB.Model(&model.Post{}).
		Where("id = ?", postID).
		Update("content", content).Error
}

func PostExists(postID uint) (bool, error) {
	_, err := FindPostByID(postID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}
