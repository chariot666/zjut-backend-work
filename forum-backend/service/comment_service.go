package service

import (
	"errors"

	"forum-backend/model"
	"forum-backend/repository"
)

func CreateComment(userID, postID uint, content string) (model.CommentResponse, error) {
	exists, err := repository.PostExists(postID)
	if err != nil {
		return model.CommentResponse{}, err
	}
	if !exists {
		return model.CommentResponse{}, errors.New("记录不存在")
	}

	comment := model.Comment{
		Content: content,
		UserID:  userID,
		PostID:  postID,
	}
	if err := repository.CreateComment(&comment); err != nil {
		return model.CommentResponse{}, err
	}

	loaded, err := repository.FindCommentByID(comment.ID)
	if err != nil {
		return model.CommentResponse{}, err
	}
	return loaded.ToResponse(), nil
}

func GetCommentsByPostID(postID uint) ([]model.CommentResponse, error) {
	comments, err := repository.FindCommentsByPostID(postID)
	if err != nil {
		return nil, err
	}
	responses := make([]model.CommentResponse, 0, len(comments))
	for _, comment := range comments {
		responses = append(responses, comment.ToResponse())
	}
	return responses, nil
}

func DeleteComment(commentID, userID uint) error {
	comment, err := repository.FindCommentByID(commentID)
	if err != nil {
		return err
	}
	if comment.UserID != userID {
		return errors.New("禁止访问")
	}
	return repository.DeleteCommentByID(commentID)
}
