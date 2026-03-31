package authz

import "manage/internal/model"

func Authorize(role int, action string) bool {
	switch role {
	case model.RoleStudent:
		return action == ActionGetMe
	case model.RoleCadre:
		return action == ActionUsersList ||
			action == ActionUsersGet ||
			action == ActionClassesList ||
			action == ActionClassesGet
	case model.RoleTeacher:
		return action == ActionUsersList ||
			action == ActionUsersGet ||
			action == ActionClassesList ||
			action == ActionClassesGet
	case model.RoleSuperAdmin:
		return true
	default:
		return false
	}
}
