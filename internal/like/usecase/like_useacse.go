package usecase

import (
	"fmt"
	"link/internal/like/entity"
	_likeRepo "link/internal/like/repository"
	_userRepo "link/internal/user/repository"
	"link/pkg/common"
	"link/pkg/dto/req"
	"net/http"
	"strings"
	"time"
)

type LikeUsecase interface {
	CreateLike(requestUserId uint, request req.LikePostRequest) error
}

type likeUsecase struct {
	userRepo _userRepo.UserRepository
	likeRepo _likeRepo.LikeRepository
}

func NewLikeUsecase(userRepo _userRepo.UserRepository,
	likeRepo _likeRepo.LikeRepository) LikeUsecase {
	return &likeUsecase{userRepo: userRepo, likeRepo: likeRepo}
}

func (u *likeUsecase) CreateLike(requestUserId uint, request req.LikePostRequest) error {

	_, err := u.userRepo.GetUserByID(requestUserId)
	if err != nil {
		fmt.Printf("사용자 조회 실패: %v", err)
		return common.NewError(http.StatusInternalServerError, "사용자 조회 실패", err)
	}

	if strings.ToUpper(request.TargetType) == "POST" {
		if request.Content == "" {
			fmt.Printf("게시물 좋아요는 내용이 필요합니다")
			return common.NewError(http.StatusBadRequest, "게시물 좋아요는 내용이 필요합니다", nil)
		}
	} else if strings.ToUpper(request.TargetType) == "COMMENT" {
		if request.Content != "" {
			fmt.Printf("댓글 좋아요는 내용이 필요없습니다")
			return common.NewError(http.StatusBadRequest, "댓글 좋아요는 내용이 필요없습니다", nil)
		}
	}

	like := &entity.Like{
		UserID:     requestUserId,
		TargetType: strings.ToUpper(request.TargetType),
		TargetID:   request.TargetID,
		Content:    request.Content,
		CreatedAt:  time.Now(),
	}

	if err := u.likeRepo.CreateLike(like); err != nil {
		fmt.Printf("좋아요 생성 실패: %v", err)
		return common.NewError(http.StatusInternalServerError, "좋아요 생성 실패", err)
	}

	return nil
}