package service

import (
	"errors"

	"forum-backend/model"
	"forum-backend/repository"
)

func CreatePost(userID uint, content string) (model.PostResponse, error) {
	post := model.Post{
		Content: content,
		UserID:  userID,
	}
	if err := repository.CreatePost(&post); err != nil {
		return model.PostResponse{}, err
	}

	loaded, err := repository.FindPostByID(post.ID)
	if err != nil {
		return model.PostResponse{}, err
	}
	return BuildPostResponse(*loaded, 0), nil
}

func ListPosts(page, pageSize int, sort string) ([]model.PostResponse, int64, error) {
	total, err := repository.CountPosts()
	if err != nil {
		return nil, 0, err
	}

	posts, err := repository.ListPosts(PostListOrder(sort), pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}

	items := make([]model.PostResponse, 0, len(posts))
	for _, post := range posts {
		commentCount, err := repository.CountComments(post.ID)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, BuildPostResponse(post, commentCount))
	}

	return items, total, nil
}

func GetPostDetail(postID uint) (model.PostDetailResponse, error) {
	post, err := repository.FindPostDetailByID(postID)
	if err != nil {
		return model.PostDetailResponse{}, err
	}

	if err := repository.IncrementViewCount(postID); err == nil {
		post.ViewCount++
	}

	comments := make([]model.CommentResponse, 0, len(post.Comments))
	for _, comment := range post.Comments {
		comments = append(comments, comment.ToResponse())
	}

	return model.PostDetailResponse{
		PostResponse: BuildPostResponse(*post, int64(len(comments))),
		Comments:     comments,
	}, nil
}

func DeletePost(userID, postID uint) error {
	post, err := repository.FindPostByID(postID)
	if err != nil {
		return err
	}
	if post.UserID != userID {
		return errors.New("禁止访问")
	}
	return repository.DeletePostWithRelatedData(postID)
}

func AdminDeletePost(postID uint) error {
	_, err := repository.FindPostByID(postID)
	if err != nil {
		return err
	}
	return repository.DeletePostWithRelatedData(postID)
}

func UpdatePost(userID, postID uint, content string) (model.PostResponse, error) {
	post, err := repository.FindPostByID(postID)
	if err != nil {
		return model.PostResponse{}, err
	}
	if post.UserID != userID {
		return model.PostResponse{}, errors.New("禁止访问")
	}
	if err := repository.UpdatePostContent(postID, content); err != nil {
		return model.PostResponse{}, err
	}

	updated, err := repository.FindPostByID(postID)
	if err != nil {
		return model.PostResponse{}, err
	}
	return BuildPostResponse(*updated, CountComments(postID)), nil
}

func BuildPostResponse(post model.Post, commentCount int64) model.PostResponse {
	return model.PostResponse{
		ID:           post.ID,
		Content:      post.Content,
		Author:       post.User.ToResponse(),
		LikeCount:    CachedLikeCount(post),
		CommentCount: commentCount,
		ViewCount:    post.ViewCount,
		CreatedAt:    post.CreatedAt,
	}
}

func PostListOrder(sort string) string {
	if sort == "hot" {
		return "((like_count * 3) + (view_count * 1) + ((SELECT COUNT(*) FROM comments WHERE comments.post_id = posts.id) * 2)) desc, created_at desc"
	}
	return "created_at desc"
}

func CountComments(postID uint) int64 {
	count, err := repository.CountComments(postID)
	if err != nil {
		return 0
	}
	return count
}
