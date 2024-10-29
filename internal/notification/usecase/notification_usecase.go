package usecase

import (
	"fmt"
	"net/http"
	"time"

	"link/internal/notification/entity"
	_notificationRepo "link/internal/notification/repository"
	_userRepo "link/internal/user/repository"
	"link/pkg/common"
	"link/pkg/dto/req"
	"link/pkg/dto/res"
)

type NotificationUsecase interface {
	GetNotifications(userId uint) ([]*entity.Notification, error)
	CreateMention(req req.NotificationRequest) (*res.CreateNotificationResponse, error)
	CreateInvite(req req.NotificationRequest) (*res.CreateNotificationResponse, error)
	CreateRequest(req req.NotificationRequest) (*res.CreateNotificationResponse, error)
	UpdateNotificationStatus(notificationId string, status string) (*res.UpdateNotificationStatusResponse, error)
}

type notificationUsecase struct {
	notificationRepo _notificationRepo.NotificationRepository
	userRepo         _userRepo.UserRepository
}

func NewNotificationUsecase(notificationRepo _notificationRepo.NotificationRepository, userRepo _userRepo.UserRepository) NotificationUsecase {
	return &notificationUsecase{notificationRepo: notificationRepo, userRepo: userRepo}
}

// TODO 알림저장 usecase 멘션 -- 수정해야함
func (n *notificationUsecase) CreateMention(req req.NotificationRequest) (*res.CreateNotificationResponse, error) {
	users, err := n.userRepo.GetUserByIds([]uint{req.SenderId, req.ReceiverId})
	if err != nil {
		return nil, common.NewError(http.StatusNotFound, "senderId 또는 receiverId가 존재하지 않습니다")
	}
	if len(users) != 2 {
		return nil, common.NewError(http.StatusNotFound, "senderId 또는 receiverId가 존재하지 않습니다")
	}

	//alarmType에 따른 처리
	var notification *entity.Notification
	notification = &entity.Notification{
		SenderId:   *users[0].ID,
		ReceiverId: *users[1].ID,
		Title:      "Mention",
		Content:    fmt.Sprintf("%s님이 %s님을 언급했습니다", *users[0].Name, *users[1].Name),
		AlarmType:  "MENTION",
		IsRead:     false,
		CreatedAt:  time.Now(),
	}

	notification, err = n.notificationRepo.CreateNotification(notification)
	if err != nil {
		return nil, common.NewError(http.StatusInternalServerError, "알림 생성에 실패했습니다")
	}

	response := &res.CreateNotificationResponse{
		ID:           notification.ID,
		SenderID:     notification.SenderId,
		ReceiverID:   notification.ReceiverId,
		Content:      notification.Content,
		AlarmType:    string(notification.AlarmType),
		InviteType:   string(notification.InviteType),
		RequestType:  string(notification.RequestType),
		CompanyId:    notification.CompanyId,
		DepartmentId: notification.DepartmentId,
		TeamId:       notification.TeamId,
		Title:        notification.Title,
		IsRead:       notification.IsRead,
		Status:       notification.Status,
		CreatedAt:    notification.CreatedAt.Format(time.DateTime),
	}

	return response, nil
}

// TODO 알림 저장 usecase -> 초대 : 초대는 어떤 초대인지 유형에 따라 분기처리
func (n *notificationUsecase) CreateInvite(req req.NotificationRequest) (*res.CreateNotificationResponse, error) {

	fmt.Println(req.SenderId, req.ReceiverId)

	users, err := n.userRepo.GetUserByIds([]uint{req.SenderId, req.ReceiverId})
	if err != nil {
		return nil, common.NewError(http.StatusNotFound, "senderId 또는 receiverId가 존재하지 않습니다")
	}

	if len(users) != 2 {
		return nil, common.NewError(http.StatusNotFound, "senderId 또는 receiverId가 존재하지 않습니다")
	}

	if users[0].Role > 3 {
		return nil, common.NewError(http.StatusBadRequest, "senderId가 관리자가 아닙니다")
	}

	if req.InviteType == "" {
		return nil, common.NewError(http.StatusBadRequest, "invite_type이 필요합니다")
	}

	notification := &entity.Notification{
		SenderId:     *users[0].ID,
		ReceiverId:   *users[1].ID,
		Title:        "INVITE",
		Content:      fmt.Sprintf("%s님이 %s님을 초대했습니다", *users[0].Name, *users[1].Name),
		AlarmType:    "INVITE",
		InviteType:   string(req.InviteType),
		CompanyId:    req.CompanyID,
		DepartmentId: req.DepartmentID,
		TeamId:       req.TeamID,
		Status:       "PENDING",
		IsRead:       false,
		CreatedAt:    time.Now(),
	}

	notification, err = n.notificationRepo.CreateNotification(notification)
	if err != nil {
		return nil, common.NewError(http.StatusInternalServerError, "알림 생성에 실패했습니다")
	}

	response := &res.CreateNotificationResponse{
		ID:           notification.ID,
		SenderID:     notification.SenderId,
		ReceiverID:   notification.ReceiverId,
		Content:      notification.Content,
		AlarmType:    string(notification.AlarmType),
		InviteType:   string(notification.InviteType),
		RequestType:  string(notification.RequestType),
		CompanyId:    notification.CompanyId,
		DepartmentId: notification.DepartmentId,
		TeamId:       notification.TeamId,
		Title:        notification.Title,
		IsRead:       notification.IsRead,
		Status:       notification.Status,
		CreatedAt:    notification.CreatedAt.Format(time.DateTime),
	}

	fmt.Println("response")
	return response, nil
}

// TODO 알림 저장 usecase -> 요청 : 요청은 어떤 요청인지 유형에 따라 분기처리
func (n *notificationUsecase) CreateRequest(req req.NotificationRequest) (*res.CreateNotificationResponse, error) {
	users, err := n.userRepo.GetUserByIds([]uint{req.SenderId, req.ReceiverId})
	if err != nil {
		return nil, common.NewError(http.StatusNotFound, "senderId 또는 receiverId가 존재하지 않습니다")
	}
	if len(users) != 2 {
		return nil, common.NewError(http.StatusNotFound, "senderId 또는 receiverId가 존재하지 않습니다")
	}

	if users[1].Role > 3 {
		return nil, common.NewError(http.StatusBadRequest, "receiverId가 관리자가 아닙니다")
	}

	if req.RequestType == "" {
		return nil, common.NewError(http.StatusBadRequest, "request_type이 필요합니다")
	}

	notification := &entity.Notification{
		SenderId:    *users[0].ID,
		ReceiverId:  *users[1].ID,
		Title:       "REQUEST",
		Content:     fmt.Sprintf("%s님이 %s님에게 요청을 보냈습니다", *users[0].Name, *users[1].Name),
		AlarmType:   "REQUEST",
		RequestType: string(req.RequestType),
		IsRead:      false,
		CreatedAt:   time.Now(),
	}

	notification, err = n.notificationRepo.CreateNotification(notification)
	if err != nil {
		return nil, common.NewError(http.StatusInternalServerError, "알림 생성에 실패했습니다")
	}

	response := &res.CreateNotificationResponse{
		ID:           notification.ID,
		SenderID:     notification.SenderId,
		ReceiverID:   notification.ReceiverId,
		Content:      notification.Content,
		AlarmType:    string(notification.AlarmType),
		InviteType:   string(notification.InviteType),
		RequestType:  string(notification.RequestType),
		CompanyId:    notification.CompanyId,
		DepartmentId: notification.DepartmentId,
		TeamId:       notification.TeamId,
		Title:        notification.Title,
		IsRead:       notification.IsRead,
		Status:       notification.Status,
		CreatedAt:    notification.CreatedAt.Format(time.DateTime),
	}

	return response, nil
}

// TODO 알림 메시지 업데이트 - 읽음 처리, 혹은 초대에 수락 등등
func (n *notificationUsecase) UpdateNotificationStatus(notificationId string, status string) (*res.UpdateNotificationStatusResponse, error) {

	//TODO string ->
	// 알림 존재 여부 확인
	notification, err := n.notificationRepo.GetNotificationByID(notificationId)
	if err != nil || notification == nil {
		return nil, common.NewError(http.StatusNotFound, "알림이 존재하지 않습니다")
	}

	if notification.Status == "ACCEPTED" || notification.Status == "REJECTED" {
		return nil, common.NewError(http.StatusBadRequest, "이미 처리된 요청입니다")
	}
	// 읽음 처리 및 상태 업데이트
	notification.IsRead = true
	if notification.AlarmType == "INVITE" || notification.AlarmType == "REQUEST" {
		notification.Status = status
	}

	notification.UpdatedAt = time.Now()

	// 데이터베이스에 업데이트 적용
	updatedNotification, err := n.notificationRepo.UpdateNotificationStatus(notification)
	if err != nil {
		return nil, common.NewError(http.StatusInternalServerError, "알림 상태 업데이트에 실패했습니다")
	}

	if status == "ACCEPTED" {
		//TODO 수락했다는 메시지
		Title := "ACCEPTED"
		Content := fmt.Sprintf("%d님이 초대를 수락했습니다", updatedNotification.ReceiverId)

		// 응답 데이터 생성 및 반환
		// TODO 응답이니 보내는 사람이 반대로 되어야함
		response := &res.UpdateNotificationStatusResponse{
			ID:         updatedNotification.ID,
			SenderID:   updatedNotification.ReceiverId,
			ReceiverID: updatedNotification.SenderId,
			Title:      Title,
			Content:    Content,
			AlarmType:  string(updatedNotification.AlarmType),
			IsRead:     updatedNotification.IsRead,
			Status:     updatedNotification.Status,
			CreatedAt:  updatedNotification.CreatedAt.Format(time.DateTime),
			UpdatedAt:  updatedNotification.UpdatedAt.Format(time.DateTime),
		}

		return response, nil
	} else if status == "REJECTED" {
		//TODO 거절했다는 메시지
		Title := "REJECTED"
		Content := fmt.Sprintf("%d님이 초대를 거절했습니다", updatedNotification.ReceiverId)

		response := &res.UpdateNotificationStatusResponse{
			ID:         updatedNotification.ID,
			SenderID:   updatedNotification.ReceiverId,
			ReceiverID: updatedNotification.SenderId,
			Title:      Title,
			Content:    Content,
			AlarmType:  string(updatedNotification.AlarmType),
			IsRead:     updatedNotification.IsRead,
			Status:     updatedNotification.Status,
			CreatedAt:  updatedNotification.CreatedAt.Format(time.DateTime),
			UpdatedAt:  updatedNotification.UpdatedAt.Format(time.DateTime),
		}

		return response, nil
	}

	return nil, nil
}

func (n *notificationUsecase) GetNotifications(userId uint) ([]*entity.Notification, error) {

	//TODO 수신자 id가 존재하는지 확인
	user, err := n.userRepo.GetUserByID(userId)
	if err != nil {
		return nil, common.NewError(http.StatusNotFound, "수신자가 존재하지 않습니다")
	}
	if user == nil {
		return nil, common.NewError(http.StatusNotFound, "수신자가 존재하지 않습니다")
	}

	//TODO 수신자 id로 알림 조회
	notifications, err := n.notificationRepo.GetNotificationsByReceiverId(*user.ID)
	if err != nil {
		return nil, common.NewError(http.StatusInternalServerError, "알림 조회에 실패했습니다")
	}

	return notifications, nil
}
