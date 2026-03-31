package authz

import (
	"manage/internal/auth"
	"manage/internal/model"
)

type Scope struct {
	SelfUserID uint
	ClassID    uint
	Grade      string
	AllowAll   bool
}

func BuildScope(a auth.Actor) Scope {
	switch a.Role {
	case model.RoleStudent:
		return Scope{SelfUserID: a.UserID}
	case model.RoleCadre:
		return Scope{ClassID: a.ClassID}
	case model.RoleTeacher:
		return Scope{ClassID: a.ClassID, Grade: a.Grade}
	case model.RoleSuperAdmin:
		return Scope{AllowAll: true}
	default:
		return Scope{}
	}
}
