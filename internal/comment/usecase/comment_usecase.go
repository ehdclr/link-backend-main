package usecase

import (
	"fmt"
	_commentRepo "link/internal/comment/repository"
	_departmentRepo "link/internal/department/repository"
	_postRepo "link/internal/post/repository"
	_userRepo "link/internal/user/repository"
	"link/pkg/common"
	"link/pkg/dto/req"
	"net/http"
	"strings"
)

type CommentUsecase interface {
	CreateComment(userId uint, req req.CommentRequest) error
}

type commentUsecase struct {
	commentRepo    _commentRepo.CommentRepository
	userRepo       _userRepo.UserRepository
	departmentRepo _departmentRepo.DepartmentRepository
	postRepo       _postRepo.PostRepository
}

func NewCommentUsecase(
	commentRepo _commentRepo.CommentRepository,
	userRepo _userRepo.UserRepository,
	departmentRepo _departmentRepo.DepartmentRepository,
	postRepo _postRepo.PostRepository,
) CommentUsecase {
	return &commentUsecase{
		commentRepo:    commentRepo,
		userRepo:       userRepo,
		postRepo:       postRepo,
		departmentRepo: departmentRepo,
	}
}

func (u *commentUsecase) CreateComment(userId uint, req req.CommentRequest) error {
	user, err := u.userRepo.GetUserByID(userId)
	if err != nil {
		fmt.Printf("사용자 조회 실패: %v", err)
		return common.NewError(http.StatusBadRequest, "사용자 조회 실패", err)
	}

	post, err := u.postRepo.GetPostByID(req.PostID)
	if err != nil {
		fmt.Printf("게시물 조회 실패: %v", err)
		return common.NewError(http.StatusBadRequest, "게시물 조회 실패", err)
	}

	if strings.ToUpper(post.Visibility) == "COMPANY" && post.CompanyID != user.UserProfile.CompanyID {
		return common.NewError(http.StatusForbidden, "회사 게시물에 대한 접근 권한이 없습니다.", nil)
	} else if strings.ToUpper(post.Visibility) == "DEPARTMENT" && post.CompanyID != user.UserProfile.CompanyID {
		//TODO 해당 게시물이 속한 부서에 사용자가 속해있는지 확인

	}

	return nil
}