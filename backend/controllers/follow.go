package controllers

import (
	"net/http"
	"strconv"

	"quietsignal/backend/models"

	"github.com/gin-gonic/gin"
)

// 关注域：关注/取消关注、用户公开主页及其统计数据

type UserProfileDTO struct {
	User        UserDTO          `json:"user"`
	Posts       []PostSummaryDTO `json:"posts"`
	PostCount   int64            `json:"postCount"`
	LikeCount   int64            `json:"likeCount"`
	Followers   int64            `json:"followers"`
	Following   int64            `json:"following"`
	FollowingMe bool             `json:"followingMe"`
}

func (cc *CommunityController) FollowUser(c *gin.Context) {
	userID := currentUserID(c)
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || uint(targetID) == userID {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不能关注自己"})
		return
	}
	var target models.User
	if err := cc.DB.Where("id = ? AND status = ?", targetID, "active").First(&target).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}
	var follow models.Follow
	if err := cc.DB.Where("follower_id = ? AND following_id = ?", userID, targetID).First(&follow).Error; err == nil {
		cc.followResponse(c, userID, uint(targetID), true)
		return
	}
	if err := cc.DB.Create(&models.Follow{FollowerID: userID, FollowingID: uint(targetID)}).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "关注失败，请重试"})
		return
	}
	// 通知被关注者，resourceId 指向关注者，便于跳转主页
	if err := createNotification(cc.DB, uint(targetID), userID, "follow", userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "关注失败，请重试"})
		return
	}
	cc.followResponse(c, userID, uint(targetID), true)
}

func (cc *CommunityController) UnfollowUser(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户不存在"})
		return
	}
	cc.DB.Where("follower_id = ? AND following_id = ?", currentUserID(c), targetID).Delete(&models.Follow{})
	cc.followResponse(c, currentUserID(c), uint(targetID), false)
}

// Profile 用户公开主页：基本资料、公开文章列表与关注数据
func (cc *CommunityController) Profile(c *gin.Context) {
	var user models.User
	if err := cc.DB.Where("username = ? AND status = ?", c.Param("username"), "active").First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}
	profile := cc.profileData(user)
	var following int64
	if currentUserID(c) > 0 {
		cc.DB.Model(&models.Follow{}).Where("follower_id = ? AND following_id = ?", currentUserID(c), user.ID).Count(&following)
	}
	profile.FollowingMe = following > 0
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": profile})
}

func (cc *CommunityController) followResponse(c *gin.Context, followerID, followingID uint, following bool) error {
	var followers int64
	cc.DB.Model(&models.Follow{}).Where("following_id = ?", followingID).Count(&followers)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"following": following, "followers": followers, "followerId": followerID}})
	return nil
}

func (cc *CommunityController) profileData(user models.User) UserProfileDTO {
	var posts []models.Post
	cc.DB.Where("author_id = ? AND status = ? AND moderation_status = ?", user.ID, "published", "normal").Preload("Tags").Preload("Author").Preload("Category").Order("published_at DESC, id DESC").Limit(30).Find(&posts)
	var postCount, likeCount, followers, following int64
	cc.DB.Model(&models.Post{}).Where("author_id = ? AND status = ? AND moderation_status = ?", user.ID, "published", "normal").Count(&postCount)
	cc.DB.Model(&models.PostLike{}).Joins("JOIN posts ON posts.id = post_likes.post_id").Where("posts.author_id = ?", user.ID).Count(&likeCount)
	cc.DB.Model(&models.Follow{}).Where("following_id = ?", user.ID).Count(&followers)
	cc.DB.Model(&models.Follow{}).Where("follower_id = ?", user.ID).Count(&following)
	items := make([]PostSummaryDTO, 0, len(posts))
	for _, post := range posts {
		items = append(items, toPostSummaryDTO(post))
	}
	return UserProfileDTO{User: toUserDTO(user), Posts: items, PostCount: postCount, LikeCount: likeCount, Followers: followers, Following: following}
}
