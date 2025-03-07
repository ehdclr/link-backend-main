package entity

import (
	"time"
)

const (
	ProjectRoleUser = iota
	ProjectMaintainer
	ProjectAdmin
	ProjectMaster
)

type Project struct {
	ID           uint          `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	StartDate    time.Time     `json:"start_date"`
	EndDate      time.Time     `json:"end_date"`
	CompanyID    uint          `json:"company_id,omitempty"`
	CreatedBy    uint          `json:"created_by"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	ProjectUsers []ProjectUser `json:"project_users"`
}

type ProjectUser struct {
	ProjectID uint `json:"project_id"`
	UserID    uint `json:"user_id"`
	Role      int  `json:"role"`
}
