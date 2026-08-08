package repository

import (
	"forum-backend/database"
	"forum-backend/model"
)

func CreateComment(comment *model.Comment) error {
	return database.DB.Create(comment).Error
}

func FindCommentByID(commentID uint) (*model.Comment, error) {
	var comment model.Comment
	if err := database.DB.Preload("User").First(&comment, commentID).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

func FindCommentsByPostID(postID uint) ([]model.Comment, error) {
	var comments []model.Comment
	if err := database.DB.
		Where("post_id = ?", postID).
		Preload("User").
		Order("created_at asc").
		Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

func DeleteCommentByID(commentID uint) error {
	return database.DB.Delete(&model.Comment{}, commentID).Error
}
