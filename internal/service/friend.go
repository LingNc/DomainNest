package service

import (
	"strconv"

	"domainnest/internal/errs"
	"domainnest/internal/model"

	"gorm.io/gorm"
)

type FriendService struct {
	db *gorm.DB
}

func NewFriendService(db *gorm.DB) *FriendService {
	return &FriendService{db: db}
}

// SendRequest creates a friend request from sender to receiver.
func (s *FriendService) SendRequest(senderID, receiverID uint64) error {
	if senderID == receiverID {
		return errs.New(errs.CannotAddSelfAsFriend, "不能添加自己为好友")
	}

	// Check receiver exists
	var receiver model.User
	if err := s.db.First(&receiver, receiverID).Error; err != nil {
		return errs.New(errs.TargetUserNotFound, "用户不存在")
	}

	// Check if already friends
	var count int64
	s.db.Model(&model.Friendship{}).Where("user_id = ? AND friend_id = ?", senderID, receiverID).Count(&count)
	if count > 0 {
		return errs.New(errs.AlreadyFriends, "已经是好友")
	}

	// Check if pending request already exists (either direction)
	var existing model.FriendRequest
	err := s.db.Where(
		"((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)) AND status = ?",
		senderID, receiverID, receiverID, senderID, "pending",
	).First(&existing).Error
	if err == nil {
		return errs.New(errs.FriendRequestPending, "好友请求已发送，请等待处理")
	}

	req := &model.FriendRequest{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Status:     "pending",
	}
	return s.db.Create(req).Error
}

// AcceptRequest accepts a pending friend request and creates bidirectional friendship.
func (s *FriendService) AcceptRequest(requestID, userID uint64) error {
	var req model.FriendRequest
	if err := s.db.First(&req, requestID).Error; err != nil {
		return errs.New(errs.RequestNotFound, "请求不存在")
	}

	if req.ReceiverID != userID {
		return errs.New(errs.NotRequestReceiver, "您不是该请求的接收者")
	}

	if req.Status != "pending" {
		return errs.New(errs.RequestAlreadyProcessed, "该请求已处理")
	}

	tx := s.db.Begin()

	// Update request status
	if err := tx.Model(&req).Update("status", "accepted").Error; err != nil {
		tx.Rollback()
		return err
	}

	// Create bidirectional friendship
	friendships := []model.Friendship{
		{UserID: req.SenderID, FriendID: req.ReceiverID},
		{UserID: req.ReceiverID, FriendID: req.SenderID},
	}
	if err := tx.Create(&friendships).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// RejectRequest rejects a pending friend request.
func (s *FriendService) RejectRequest(requestID, userID uint64) error {
	var req model.FriendRequest
	if err := s.db.First(&req, requestID).Error; err != nil {
		return errs.New(errs.RequestNotFound, "请求不存在")
	}

	if req.ReceiverID != userID {
		return errs.New(errs.NotRequestReceiver, "您不是该请求的接收者")
	}

	if req.Status != "pending" {
		return errs.New(errs.RequestAlreadyProcessed, "该请求已处理")
	}

	return s.db.Model(&req).Update("status", "rejected").Error
}

// RemoveFriend removes a bidirectional friendship.
func (s *FriendService) RemoveFriend(userID, friendID uint64) error {
	result := s.db.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
		userID, friendID, friendID, userID).Delete(&model.Friendship{})
	if result.RowsAffected == 0 {
		return errs.New(errs.FriendRelationNotFound, "好友关系不存在")
	}
	return result.Error
}

// ListFriends returns all friends of a user.
func (s *FriendService) ListFriends(userID uint64) ([]model.Friendship, error) {
	var friendships []model.Friendship
	err := s.db.Preload("Friend").Where("user_id = ?", userID).Order("created_at DESC").Find(&friendships).Error
	return friendships, err
}

// ListPendingRequests returns pending friend requests received by the user.
func (s *FriendService) ListPendingRequests(userID uint64) ([]model.FriendRequest, error) {
	var requests []model.FriendRequest
	err := s.db.Preload("Sender").Where("receiver_id = ? AND status = ?", userID, "pending").
		Order("created_at DESC").Find(&requests).Error
	return requests, err
}

// ListSentRequests returns pending friend requests sent by the user.
func (s *FriendService) ListSentRequests(userID uint64) ([]model.FriendRequest, error) {
	var requests []model.FriendRequest
	err := s.db.Preload("Receiver").Where("sender_id = ? AND status = ?", userID, "pending").
		Order("created_at DESC").Find(&requests).Error
	return requests, err
}

// SearchUsers searches users by username or nickname, excluding self and existing friends.
func (s *FriendService) SearchUsers(userID uint64, keyword string) ([]model.User, error) {
	if len(keyword) < 2 {
		return nil, errs.New(errs.SearchKeywordTooShort, "搜索关键词太短")
	}

	// Get friend IDs
	var friendIDs []uint64
	s.db.Model(&model.Friendship{}).Where("user_id = ?", userID).Pluck("friend_id", &friendIDs)

	var users []model.User
	query := s.db.Where("(username LIKE ? OR nickname LIKE ?) AND id != ?",
		"%"+keyword+"%", "%"+keyword+"%", userID)

	if len(friendIDs) > 0 {
		query = query.Where("id NOT IN ?", friendIDs)
	}

	err := query.Select("id", "username", "nickname", "avatar").Limit(20).Find(&users).Error
	return users, err
}

// SearchAllUsers searches all users by username or nickname, with friends sorted first.
func (s *FriendService) SearchAllUsers(userID uint64, keyword string) ([]model.User, error) {
	if len(keyword) < 2 {
		return nil, errs.New(errs.SearchKeywordTooShort, "搜索关键词太短")
	}

	var users []model.User
	query := s.db.
		Select("users.id, users.username, users.nickname, users.avatar").
		Joins("LEFT JOIN friendships ON friendships.user_id = ? AND friendships.friend_id = users.id", userID)

	if id, err := strconv.ParseUint(keyword, 10, 64); err == nil {
		query = query.Where("(users.username LIKE ? OR users.nickname LIKE ? OR users.id = ?)", "%"+keyword+"%", "%"+keyword+"%", id)
	} else {
		query = query.Where("(users.username LIKE ? OR users.nickname LIKE ?)", "%"+keyword+"%", "%"+keyword+"%")
	}

	err := query.Order("CASE WHEN friendships.user_id IS NOT NULL THEN 0 ELSE 1 END, users.username").
		Limit(20).
		Find(&users).Error
	return users, err
}

// AreFriends checks if two users are friends.
func (s *FriendService) AreFriends(userID, otherID uint64) bool {
	var count int64
	s.db.Model(&model.Friendship{}).Where("user_id = ? AND friend_id = ?", userID, otherID).Count(&count)
	return count > 0
}
