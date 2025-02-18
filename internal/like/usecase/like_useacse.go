package usecase

import (
	"encoding/json"
	"fmt"
	_commentRepo "link/internal/comment/repository"
	"link/internal/like/entity"
	_likeRepo "link/internal/like/repository"
	_postRepo "link/internal/post/repository"
	_userRepo "link/internal/user/repository"
	"link/pkg/common"
	"link/pkg/dto/req"
	"link/pkg/dto/res"
	_nats "link/pkg/nats"
	"net/http"
	"strings"
	"time"
)

type LikeUsecase interface {
	CreatePostLike(requestUserId uint, request req.LikePostRequest) error
	GetPostLikeList(requestUserId uint, postId uint) ([]*res.GetPostLikeListResponse, error)
	DeletePostLike(requestUserId uint, postId uint, emojiId uint) error
	CreateCommentLike(requestUserId uint, commentId uint) error
	DeleteCommentLike(requestUserId uint, commentId uint) error
}

type likeUsecase struct {
	userRepo      _userRepo.UserRepository
	likeRepo      _likeRepo.LikeRepository
	postRepo      _postRepo.PostRepository
	commentRepo   _commentRepo.CommentRepository
	natsPublisher *_nats.NatsPublisher
}

func NewLikeUsecase(userRepo _userRepo.UserRepository,
	likeRepo _likeRepo.LikeRepository,
	postRepo _postRepo.PostRepository,
	commentRepo _commentRepo.CommentRepository,
	natsPublisher *_nats.NatsPublisher) LikeUsecase {
	return &likeUsecase{
		userRepo:      userRepo,
		likeRepo:      likeRepo,
		postRepo:      postRepo,
		commentRepo:   commentRepo,
		natsPublisher: natsPublisher,
	}
}

// TODO 게시글 이모지 좋아요
func (u *likeUsecase) CreatePostLike(requestUserId uint, request req.LikePostRequest) error {

	_, err := u.userRepo.GetUserByID(requestUserId)
	if err != nil {
		fmt.Printf("사용자 조회 실패: %v", err)
		return &common.AppError{
			StatusCode: http.StatusInternalServerError,
			Message:    "사용자 조회 실패",
			Err:        err,
		}
	}

	post, err := u.postRepo.GetPostByID(request.TargetID)
	if err != nil {
		fmt.Printf("해당 게시물이 존재하지 않습니다: %v", err)
		return &common.AppError{
			StatusCode: http.StatusInternalServerError,
			Message:    "해당 게시물이 존재하지 않습니다",
			Err:        err,
		}
	}

	if strings.ToUpper(request.TargetType) != "POST" {
		fmt.Printf("이모지 좋아요 대상이 올바르지 않습니다")
		return &common.AppError{
			StatusCode: http.StatusBadRequest,
			Message:    "이모지 좋아요 대상이 올바르지 않습니다",
			Err:        nil,
		}
	}

	if request.Content == "" {
		fmt.Printf("이모지가 없습니다")
		return &common.AppError{
			StatusCode: http.StatusBadRequest,
			Message:    "이모지가 없습니다",
			Err:        nil,
		}
	}

	like := &entity.Like{
		UserID:     requestUserId,
		TargetType: strings.ToUpper(request.TargetType),
		TargetID:   request.TargetID,
		Unified:    request.Unified,
		Content:    request.Content,
		CreatedAt:  time.Now(),
	}

	if err := u.likeRepo.CreatePostLike(like); err != nil {
		fmt.Printf("좋아요 생성 실패: %v", err)
		return &common.AppError{
			StatusCode: http.StatusInternalServerError,
			Message:    "좋아요 생성 실패",
			Err:        err,
		}
	}

	//TODO 구조는 notification 패키지에 맞춰서 변경
	notification := map[string]interface{}{
		"alarm_type":  "LIKE",
		"sender_id":   like.UserID,
		"receiver_id": post.UserID,
		"post_id":     post.ID,
		"created_at":  like.CreatedAt,
	}
	notificationJson, err := json.Marshal(notification)
	if err != nil {
		fmt.Printf("알림 생성 실패: %v", err)
		return &common.AppError{
			StatusCode: http.StatusInternalServerError,
			Message:    "알림 생성 실패",
			Err:        err,
		}
	}

	//TODO nats pub으로 해당 게시글 주인에게 알림 전송 로그성 데이터는 mongodb에 저장
	u.natsPublisher.PublishEvent("like.post.created", notificationJson)
	return nil
}

// TODO 게시글 이모지 취소 -> 좋아요 삭제
func (u *likeUsecase) DeletePostLike(requestUserId uint, postId uint, emojiId uint) error {

	//TODO 해당 게시물에 대한 이모지가 있는지 확인
	like, err := u.likeRepo.GetPostLikeByID(requestUserId, postId, emojiId)
	if err != nil {
		fmt.Printf("해당 사용자가 좋아요를 해당 이모지를 누른 적이 없습니다: %v", err)
		return &common.AppError{
			StatusCode: http.StatusInternalServerError,
			Message:    "해당 사용자가 좋아요를 해당 이모지를 누른 적이 없습니다",
			Err:        err,
		}
	}

	if err := u.likeRepo.DeletePostLike(like.ID); err != nil {
		fmt.Printf("좋아요 삭제 실패: %v", err)
		return &common.AppError{
			StatusCode: http.StatusInternalServerError,
			Message:    "좋아요 삭제 실패",
			Err:        err,
		}
	}

	return nil
}

func (u *likeUsecase) GetPostLikeList(requestUserId uint, postId uint) ([]*res.GetPostLikeListResponse, error) {
	likeList, err := u.likeRepo.GetPostLikeList(requestUserId, postId)
	if err != nil {
		fmt.Printf("게시물 좋아요 조회 실패: %v", err)
		return nil, &common.AppError{
			StatusCode: http.StatusInternalServerError,
			Message:    "게시물 좋아요 조회 실패",
			Err:        err,
		}
	}

	response := make([]*res.GetPostLikeListResponse, len(likeList))
	for i, like := range likeList {
		response[i] = &res.GetPostLikeListResponse{
			TargetType: "POST",
			TargetID:   like.TargetID,
			EmojiId:    like.EmojiID,
			Unified:    like.Unified,
			Content:    like.Content,
			//TODO 본인이 해당 이모지를 눌렀는지 확인하는 필드 (다른사람이 추가한걸	확인하는것)
			IsCliked: like.IsCliked,
			Count:    int(like.Count),
		}
	}

	return response, nil
}

func (u *likeUsecase) CreateCommentLike(requestUserId uint, commentId uint) error {

	_, err := u.userRepo.GetUserByID(requestUserId)
	if err != nil {
		fmt.Printf("해당 사용자가 존재하지 않습니다: %v", err)
		return &common.AppError{
			StatusCode: http.StatusNotFound,
			Message:    "해당 사용자가 존재하지 않습니다",
			Err:        err,
		}
	}

	_, err = u.commentRepo.GetCommentByID(commentId)
	if err != nil {
		fmt.Printf("해당 댓글이 존재하지 않습니다: %v", err)
		return &common.AppError{
			StatusCode: http.StatusNotFound,
			Message:    "해당 댓글이 존재하지 않습니다",
			Err:        err,
		}
	}

	like := &entity.Like{
		UserID:     requestUserId,
		TargetType: "COMMENT",
		TargetID:   commentId,
		CreatedAt:  time.Now(),
	}

	if err := u.likeRepo.CreateCommentLike(like); err != nil {
		fmt.Printf("좋아요 생성 실패: %v", err.Error())
		return &common.AppError{
			StatusCode: http.StatusInternalServerError,
			Message:    "좋아요 생성 실패",
			Err:        err,
		}
	}

	return nil
}

func (u *likeUsecase) DeleteCommentLike(requestUserId uint, commentId uint) error {
	_, err := u.userRepo.GetUserByID(requestUserId)
	if err != nil {
		fmt.Printf("사용자 조회 실패: %v", err)
		return &common.AppError{
			StatusCode: http.StatusNotFound,
			Message:    "사용자 조회 실패",
			Err:        err,
		}
	}

	_, err = u.commentRepo.GetCommentByID(commentId)
	if err != nil {
		fmt.Printf("해당 댓글이 존재하지 않습니다: %v", err)
		return &common.AppError{
			StatusCode: http.StatusNotFound,
			Message:    "해당 댓글이 존재하지 않습니다",
			Err:        err,
		}
	}

	like, err := u.likeRepo.GetCommentLikeByID(requestUserId, commentId)
	if err != nil {
		fmt.Printf("해당 사용자가 해당 댓글에 좋아요를 누른 적이 없습니다: %v", err)
		return &common.AppError{
			StatusCode: http.StatusNotFound,
			Message:    "해당 사용자가 해당 댓글에 좋아요를 누른 적이 없습니다",
			Err:        err,
		}
	}

	if err := u.likeRepo.DeleteCommentLike(like.ID); err != nil {
		fmt.Printf("좋아요 삭제 실패: %v", err)
		return &common.AppError{
			StatusCode: http.StatusInternalServerError,
			Message:    "좋아요 삭제 실패",
			Err:        err,
		}
	}

	return nil
}
