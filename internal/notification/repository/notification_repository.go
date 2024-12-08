package repository

import (
	"link/internal/notification/entity"
)

type NotificationRepository interface {
	CreateNotification(notification *entity.Notification) (*entity.Notification, error)

	GetNotificationsByReceiverId(receiverId uint) ([]*entity.Notification, error)
	GetNotificationByID(notificationId string) (*entity.Notification, error)

	UpdateNotificationStatus(notification *entity.Notification) (*entity.Notification, error)
	UpdateNotificationReadStatus(notification *entity.Notification) (*entity.Notification, error)
}
