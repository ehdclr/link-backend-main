package usecase

import (
	"fmt"
	"log"
	"net/http"

	_companyRepo "link/internal/company/repository"
	_userEntity "link/internal/user/entity"
	_userRepo "link/internal/user/repository"
	"link/pkg/common"
	"link/pkg/dto/res"
)

type CompanyUsecase interface {
	GetAllCompanies() ([]res.GetCompanyInfoResponse, error)
	GetCompanyInfo(id uint) (*res.GetCompanyInfoResponse, error)
	SearchCompany(companyName string) ([]res.GetCompanyInfoResponse, error)

	AddUserToCompany(requestUserId uint, userId uint, companyId uint) error
	GetOrganizationByCompany(requestUserId uint) (*res.OrganizationResponse, error)
}

type companyUsecase struct {
	companyRepository _companyRepo.CompanyRepository
	userRepository    _userRepo.UserRepository
}

func NewCompanyUsecase(companyRepository _companyRepo.CompanyRepository, userRepository _userRepo.UserRepository) CompanyUsecase {
	return &companyUsecase{companyRepository: companyRepository, userRepository: userRepository}
}

// TODO 회사 전체 목록 조회
func (u *companyUsecase) GetAllCompanies() ([]res.GetCompanyInfoResponse, error) {
	companies, err := u.companyRepository.GetAllCompanies()
	if err != nil {
		return nil, common.NewError(http.StatusInternalServerError, "서버 에러", err)
	}

	response := make([]res.GetCompanyInfoResponse, len(companies))
	for i, company := range companies {
		response[i] = res.GetCompanyInfoResponse{
			ID:                    company.ID,
			CpName:                company.CpName,
			CpLogo:                company.CpLogo,
			RepresentativeName:    company.RepresentativeName,
			RepresentativeTel:     company.RepresentativePhoneNumber,
			RepresentativeEmail:   company.RepresentativeEmail,
			RepresentativeAddress: company.RepresentativeAddress,
		}
	}

	return response, nil
}

// TODO 회사 조회
func (u *companyUsecase) GetCompanyInfo(id uint) (*res.GetCompanyInfoResponse, error) {
	company, err := u.companyRepository.GetCompanyByID(id)
	if err != nil {
		return nil, common.NewError(http.StatusInternalServerError, "서버 에러", err)
	}

	response := &res.GetCompanyInfoResponse{
		ID:                    company.ID,
		CpName:                company.CpName,
		CpLogo:                company.CpLogo,
		RepresentativeName:    company.RepresentativeName,
		RepresentativeTel:     company.RepresentativePhoneNumber,
		RepresentativeEmail:   company.RepresentativeEmail,
		RepresentativeAddress: company.RepresentativeAddress,
	}

	return response, nil
}

// TODO 회사 검색
func (u *companyUsecase) SearchCompany(companyName string) ([]res.GetCompanyInfoResponse, error) {
	companies, err := u.companyRepository.SearchCompany(companyName)
	if err != nil {
		return nil, common.NewError(http.StatusInternalServerError, "서버 에러", err)
	}

	response := make([]res.GetCompanyInfoResponse, len(companies))
	for i, company := range companies {
		response[i] = res.GetCompanyInfoResponse{
			ID:                    company.ID,
			CpName:                company.CpName,
			CpLogo:                company.CpLogo,
			RepresentativeName:    company.RepresentativeName,
			RepresentativeTel:     company.RepresentativePhoneNumber,
			RepresentativeEmail:   company.RepresentativeEmail,
			RepresentativeAddress: company.RepresentativeAddress,
		}
	}

	return response, nil
}

// TODO 회사에 사용자 추가
func (u *companyUsecase) AddUserToCompany(requestUserId uint, userId uint, companyId uint) error {
	//TODO requestUserId의 Role이 3이상이여야하고 3이라면, 자기 회사만 가능

	adminUser, err := u.userRepository.GetUserByID(requestUserId)
	if err != nil {
		return common.NewError(http.StatusInternalServerError, "서버 에러", err)
	}
	if adminUser.Role > _userEntity.RoleCompanySubManager {
		log.Println("권한이 없습니다")
		return common.NewError(http.StatusForbidden, "권한이 없습니다", err)
	}

	user, err := u.userRepository.GetUserByID(userId)
	if err != nil {
		return common.NewError(http.StatusInternalServerError, "서버 에러", err)
	}

	if user.UserProfile.CompanyID != nil {
		log.Println("이미 회사에 소속된 사용자입니다")
		return common.NewError(http.StatusBadRequest, "이미 회사에 소속된 사용자입니다", err)
	}

	//TODO 만약에 Role이 3이라면 자기 회사만 사용자 추가 가능
	if *adminUser.UserProfile.CompanyID != companyId && adminUser.Role > _userEntity.RoleCompanySubManager {
		log.Println("권한이 없습니다")
		return common.NewError(http.StatusForbidden, "권한이 없습니다", err)
	}

	//TODO 사용자 companyId 업데이트
	err = u.userRepository.UpdateUser(userId, nil, map[string]interface{}{"company_id": companyId})
	if err != nil {
		return common.NewError(http.StatusInternalServerError, "서버 에러", err)
	}

	return nil
}

//TODO 회사에 사용자 삭제

//TODO 회사 구독 취소 (회사 관리자만 - 자기 회사 구독 취소 가능)

// TODO 회사 조직도 조회
// TODO 자기가 속한 회사 조직도 - 수정중
func (u *companyUsecase) GetOrganizationByCompany(requestUserId uint) (*res.OrganizationResponse, error) {
	user, err := u.userRepository.GetUserByID(requestUserId)
	if err != nil {
		fmt.Printf("사용자 조회에 실패했습니다: %v", err)
		return nil, common.NewError(http.StatusInternalServerError, "사용자 조회에 실패했습니다", err)
	}

	// CompanyID가 nil인지 확인
	if user.UserProfile.CompanyID == nil {
		log.Printf("사용자가 소속된 회사가 없습니다: 사용자 ID %d", requestUserId)
		return nil, common.NewError(http.StatusBadRequest, "사용자가 소속된 회사가 없습니다", nil)
	}

	companyId := *user.UserProfile.CompanyID
	companyName := (*user.UserProfile.Company)["name"].(string)

	// 회사에 속한 모든 사용자 조회
	users, err := u.userRepository.GetUsersByCompany(companyId, nil)
	if err != nil {
		fmt.Printf("회사 사용자 조회에 실패했습니다: %v", err)
		return nil, common.NewError(http.StatusInternalServerError, "회사 사용자 조회에 실패했습니다", err)
	}

	// 부서별 사용자 분류
	departmentMap := make(map[uint]*res.OrganizationDepartmentInfoResponse)
	var unassignedUsers []res.GetOrganizationUserInfoResponse

	for _, user := range users {
		// 각 사용자의 Position 정보 가져오기
		positionName := ""
		if user.UserProfile.Position != nil {
			if posName, ok := (*user.UserProfile.Position)["name"].(string); ok {
				positionName = posName
			}
		}

		var positionId uint
		if user.UserProfile.PositionId != nil {
			positionId = *user.UserProfile.PositionId
		}

		// 사용자가 소속된 부서가 있는 경우와 없는 경우 처리
		if len(user.UserProfile.Departments) > 0 {
			for _, dept := range user.UserProfile.Departments {
				if deptID, ok := (*dept)["id"].(uint); ok {
					deptName := ""
					if name, ok := (*dept)["name"].(string); ok {
						deptName = name
					}

					// 부서가 이미 맵에 없다면 새로 생성
					if _, exists := departmentMap[deptID]; !exists {
						departmentMap[deptID] = &res.OrganizationDepartmentInfoResponse{
							DepartmentId:   deptID,
							DepartmentName: deptName,
							Users:          []res.GetOrganizationUserInfoResponse{},
						}
					}

					// 해당 부서에 사용자 추가
					departmentMap[deptID].Users = append(departmentMap[deptID].Users, res.GetOrganizationUserInfoResponse{
						ID:           *user.ID,
						Email:        *user.Email,
						Name:         *user.Name,
						Role:         uint(user.Role),
						Phone:        *user.Phone,
						Nickname:     *user.Nickname,
						PositionId:   positionId,
						PositionName: positionName,
						EntryDate:    user.UserProfile.EntryDate,
					})
				}
			}
		} else {
			// 부서가 없는 사용자라면 unassignedUsers에 추가
			unassignedUsers = append(unassignedUsers, res.GetOrganizationUserInfoResponse{
				ID:           *user.ID,
				Email:        *user.Email,
				Name:         *user.Name,
				Role:         uint(user.Role),
				Phone:        *user.Phone,
				Nickname:     *user.Nickname,
				PositionId:   positionId,
				PositionName: positionName,
				EntryDate:    user.UserProfile.EntryDate,
			})
		}
	}

	// 부서 정보를 배열로 구성
	var departments []res.OrganizationDepartmentInfoResponse
	for _, dept := range departmentMap {
		departments = append(departments, *dept)
	}

	// 최종 응답 구조체 생성
	organizationResponse := &res.OrganizationResponse{
		CompanyId:       companyId,
		CompanyName:     companyName,
		Departments:     departments,
		UnassignedUsers: unassignedUsers, // 부서 없는 사용자 리스트 추가
	}

	return organizationResponse, nil

}
