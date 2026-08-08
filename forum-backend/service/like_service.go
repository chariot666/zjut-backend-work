package service

import (
	"fmt"
	"strconv"
	"time"

	"forum-backend/database"
	"forum-backend/model"
	"forum-backend/repository"
)

type LikeStatus struct {
	PostID uint `json:"post_id"`
	Liked  bool `json:"liked"`
}

func ToggleLike(userID, postID uint) (bool, error) {
	exists, err := repository.PostExists(postID)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, fmt.Errorf("记录不存在")
	}

	isLiked, err := repository.ToggleLike(userID, postID)
	if err != nil {
		return false, err
	}

	post, err := repository.FindPostByID(postID)
	if err == nil {
		syncLikeCache(*post, userID, isLiked)
	}

	return isLiked, nil
}

func GetLikeStatus(userID uint, postIDs []uint) ([]LikeStatus, error) {
	if database.RDB != nil {
		status, err := getLikeStatusFromRedis(userID, postIDs)
		if err == nil {
			return status, nil
		}
	}

	likes, err := repository.FindLikesByUserAndPosts(userID, postIDs)
	if err != nil {
		return nil, err
	}

	likedSet := make(map[uint]bool, len(likes))
	for _, like := range likes {
		likedSet[like.PostID] = true
	}

	status := make([]LikeStatus, 0, len(postIDs))
	for _, postID := range postIDs {
		status = append(status, LikeStatus{
			PostID: postID,
			Liked:  likedSet[postID],
		})
	}
	return status, nil
}

func syncLikeCache(post model.Post, userID uint, isLiked bool) {
	if database.RDB == nil {
		return
	}

	userIDStr := strconv.FormatUint(uint64(userID), 10)
	likesKey := postLikesKey(post.ID)
	if isLiked {
		database.RDB.SRem(database.Ctx, likesKey, "__empty__")
		database.RDB.SAdd(database.Ctx, likesKey, userIDStr)
	} else {
		database.RDB.SRem(database.Ctx, likesKey, userIDStr)
		if post.LikeCount == 0 {
			database.RDB.SAdd(database.Ctx, likesKey, "__empty__")
		}
	}
	database.RDB.Set(database.Ctx, postLikeCountKey(post.ID), post.LikeCount, time.Hour)
	database.RDB.Expire(database.Ctx, likesKey, time.Hour)
}

func getLikeStatusFromRedis(userID uint, postIDs []uint) ([]LikeStatus, error) {
	status := make([]LikeStatus, 0, len(postIDs))
	userIDStr := strconv.FormatUint(uint64(userID), 10)

	for _, postID := range postIDs {
		if err := warmLikeCache(postID); err != nil {
			return nil, err
		}

		liked, err := database.RDB.SIsMember(database.Ctx, postLikesKey(postID), userIDStr).Result()
		if err != nil {
			return nil, err
		}
		status = append(status, LikeStatus{
			PostID: postID,
			Liked:  liked,
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

	post, err := repository.FindPostByID(postID)
	if err != nil {
		return err
	}

	likes, err := repository.FindLikesByPostID(postID)
	if err != nil {
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

func CachedLikeCount(post model.Post) int {
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
