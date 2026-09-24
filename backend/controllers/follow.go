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
		if err := cc.followResponse(c, userID, uint(targetID), true); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取关注数据失败"})
		}
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
	if err := cc.followResponse(c, userID, uint(targetID), true); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取关注数据失败"})
	}
}

func (cc *CommunityController) UnfollowUser(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户不存在"})
		return
	}
	if err := cc.DB.Where("follower_id = ? AND following_id = ?", currentUserID(c), targetID).Delete(&models.Follow{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "取消关注失败"})
		return
	}
	if err := cc.followResponse(c, currentUserID(c), uint(targetID), false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取关注数据失败"})
	}
}

// Profile 用户公开主页：基本资料、公开文章列表（分页）与关注数据。
// 文章列表曾经硬编码 Limit 30，高产作者的主页会静默丢掉更早的文章
func (cc *CommunityController) Profile(c *gin.Context) {
	var user models.User
	if err := cc.DB.Where("username = ? AND status = ?", c.Param("username"), "active").First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}
	page, pageSize := pagination(c)
	profile, err := cc.profileData(user, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取用户资料失败"})
		return
	}
	var following int64
	if currentUserID(c) > 0 {
		if err := cc.DB.Model(&models.Follow{}).Where("follower_id = ? AND following_id = ?", currentUserID(c), user.ID).Count(&following).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取关注数据失败"})
			return
		}
	}
	profile.FollowingMe = following > 0
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": profile})
}

func (cc *CommunityController) followResponse(c *gin.Context, followerID, followingID uint, following bool) error {
	var followers int64
	if err := cc.DB.Model(&models.Follow{}).Where("following_id = ?", followingID).Count(&followers).Error; err != nil {
		return err
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"following": following, "followers": followers, "followerId": followerID}})
	return nil
}

func (cc *CommunityController) profileData(user models.User, page, pageSize int) (UserProfileDTO, error) {
	var posts []models.Post
	if err := cc.DB.Where("author_id = ? AND status = ? AND moderation_status = ?", user.ID, "published", "normal").Preload("Tags").Preload("Author").Preload("Category").Order("published_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error; err != nil {
		return UserProfileDTO{}, err
	}
	var postCount, likeCount, followers, following int64
	if err := cc.DB.Model(&models.Post{}).Where("author_id = ? AND status = ? AND moderation_status = ?", user.ID, "published", "normal").Count(&postCount).Error; err != nil {
		return UserProfileDTO{}, err
	}
	if err := cc.DB.Model(&models.PostLike{}).Joins("JOIN posts ON posts.id = post_likes.post_id").Where("posts.author_id = ?", user.ID).Count(&likeCount).Error; err != nil {
		return UserProfileDTO{}, err
	}
	if err := cc.DB.Model(&models.Follow{}).Where("following_id = ?", user.ID).Count(&followers).Error; err != nil {
		return UserProfileDTO{}, err
	}
	if err := cc.DB.Model(&models.Follow{}).Where("follower_id = ?", user.ID).Count(&following).Error; err != nil {
		return UserProfileDTO{}, err
	}
	items := make([]PostSummaryDTO, 0, len(posts))
	for _, post := range posts {
		items = append(items, toPostSummaryDTO(post))
	}
	return UserProfileDTO{User: toUserDTO(user), Posts: items, PostCount: postCount, LikeCount: likeCount, Followers: followers, Following: following}, nil
}
