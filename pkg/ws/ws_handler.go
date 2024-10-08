package ws

import (
	"link/internal/chat/usecase"
	"link/pkg/dto/req"
	"link/pkg/util"
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WsHandler struct는 WebSocketHub와 연동합니다.
type WsHandler struct {
	hub         *WebSocketHub
	chatUsecase usecase.ChatUsecase
}

// NewWsHandler는 WebSocketHub를 받아서 새로운 WsHandler를 반환합니다.
func NewWsHandler(hub *WebSocketHub, chatUsecase usecase.ChatUsecase) *WsHandler {
	return &WsHandler{
		hub:         hub,
		chatUsecase: chatUsecase,
	}
}

// HandleWebSocket 함수는 WebSocket 요청을 처리합니다.
func (h *WsHandler) HandleWebSocket(c *gin.Context) {
	// WebSocket 연결을 업그레이드합니다.
	conn, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket 업그레이드 실패: %v", err)
		return
	}
	defer conn.Close()

	// 첫 번째 메시지에서 토큰을 받아서 처리
	var initialMessage struct {
		Token  string `json:"token"`
		RoomID string `json:"roomId"`
	}
	err = conn.ReadJSON(&initialMessage)
	if err != nil {
		log.Printf("초기 메시지 수신 실패: %v", err)
		conn.WriteMessage(websocket.CloseMessage, []byte("Invalid initial message"))
		return
	}

	if initialMessage.Token == "" {
		conn.WriteMessage(websocket.CloseMessage, []byte("Unauthorized"))
		return
	}
	tokenString := strings.TrimPrefix(initialMessage.Token, "Bearer ")
	// Access Token을 검증
	claims, err := util.ValidateAccessToken(tokenString)
	if err != nil {
		log.Printf("토큰 검증 실패: %v", err)
		conn.WriteMessage(websocket.CloseMessage, []byte("Unauthorized"))
		return
	}

	requestUserId := claims.UserId

	// roomId를 uint로 변환
	roomId, err := strconv.ParseUint(initialMessage.RoomID, 10, 32) // uint32 범위로 변환
	if err != nil {
		conn.WriteMessage(websocket.CloseMessage, []byte("Invalid roomId"))
		return
	}

	// TODO: DB에서 roomId가 실제로 존재하는지 확인 (chatUsecase를 통해 DB 검증)
	chatRoom, err := h.chatUsecase.GetChatRoomById(uint(roomId))
	if err != nil || chatRoom == nil {
		log.Printf("존재하지 않는 채팅방 ID: %d", roomId)
		conn.WriteMessage(websocket.CloseMessage, []byte("Invalid roomId"))
		return
	}

	// WebSocket 클라이언트를 등록합니다.
	h.hub.RegisterClient(conn, requestUserId)
	defer h.hub.UnregisterClient(conn, requestUserId)

	// WebSocket 메시지를 계속해서 수신하고 처리합니다.
	for {
		var message req.SendMessageRequest
		err := conn.ReadJSON(&message)
		if err != nil {
			log.Printf("메시지 수신 실패: %v", err)
			break
		}

		// chat_id를 기준으로 메시지 전송
		h.hub.SendMessageToChatRoom(uint(roomId), message)
	}
}
