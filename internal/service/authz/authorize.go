package authz

import "manage/internal/model"

func Authorize(role int, action string) bool {
	switch role {
	case model.RoleStudent:
		return action == ActionGetMe || action == ActionKnowledgeSearch
	case model.RoleCadre:
		return action == ActionUsersList ||
			action == ActionUsersGet ||
			action == ActionKnowledgeSearch ||
			action == ActionKnowledgeList ||
			action == ActionKnowledgeCreate ||
			action == ActionKnowledgePatch ||
			action == ActionClassesList ||
			action == ActionClassesGet
	case model.RoleTeacher:
		return action == ActionUsersList ||
			action == ActionUsersGet ||
			action == ActionKnowledgeSearch ||
			action == ActionKnowledgeList ||
			action == ActionKnowledgeCreate ||
			action == ActionKnowledgePatch ||
			action == ActionClassesList ||
			action == ActionClassesGet
	case model.RoleSuperAdmin:
		return true
	default:
		return false
	}
}
