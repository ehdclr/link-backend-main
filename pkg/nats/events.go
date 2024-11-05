package nats

// ! 이벤트 정의 패키지
const (
	UserCreatedEvent = "user.created"
	UserUpdatedEvent = "user.updated"
	UserDeletedEvent = "user.deleted"

	CompanyCreatedEvent = "company.created"
	CompanyUpdatedEvent = "company.updated"
	CompanyDeletedEvent = "company.deleted"

	RoleCreatedEvent = "role.created"
	RoleUpdatedEvent = "role.updated"
	RoleDeletedEvent = "role.deleted"

	//채팅방 참가
	ChatRoomJoinedEvent = "chat_room.joined"
	ChatRoomLeftEvent   = "chat_room.left"
	//채팅 메시지
	ChatMessageSentEvent = "chat_message.sent"

	//채팅방 관련
	ChatRoomCreatedEvent = "chat_room.created"
	ChatRoomUpdatedEvent = "chat_room.updated"
	ChatRoomDeletedEvent = "chat_room.deleted"
)
