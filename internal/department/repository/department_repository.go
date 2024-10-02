package repository

import "link/internal/department/entity"

type DepartmentRepository interface {
	CreateDepartment(department *entity.Department) error
	GetDepartments() ([]entity.Department, error)
	GetDepartment(departmentID uint) (*entity.Department, error)
	UpdateDepartment(departmentID uint, department *entity.Department) (*entity.Department, error)
	DeleteDepartment(departmentID uint) error
}
