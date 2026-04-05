package authz

import "manage/internal/model"

func Authorize(role int, action string) bool {
	switch role {
	case model.RoleStudent:
		return action == ActionGetMe ||
			action == ActionMePatch ||
			action == ActionProfileHomeGet ||
			action == ActionKnowledgeSearch ||
			action == ActionNotifUnreadGet ||
			action == ActionFilesUpload ||
			action == ActionFilesGet ||
			action == ActionFilesList
	case model.RoleCadre:
		return action == ActionUsersList ||
			action == ActionUsersGet ||
			action == ActionGetMe ||
			action == ActionMePatch ||
			action == ActionProfileHomeGet ||
			action == ActionKnowledgeSearch ||
			action == ActionNotifUnreadGet ||
			action == ActionKnowledgeList ||
			action == ActionKnowledgeGet ||
			action == ActionKnowledgeCreate ||
			action == ActionKnowledgePatch ||
			action == ActionClassesList ||
			action == ActionClassesGet ||
			action == ActionFilesUpload ||
			action == ActionFilesGet ||
			action == ActionFilesList
	case model.RoleTeacher:
		return action == ActionUsersList ||
			action == ActionUsersGet ||
			action == ActionGetMe ||
			action == ActionMePatch ||
			action == ActionProfileHomeGet ||
			action == ActionKnowledgeSearch ||
			action == ActionNotifUnreadGet ||
			action == ActionKnowledgeList ||
			action == ActionKnowledgeGet ||
			action == ActionKnowledgeCreate ||
			action == ActionKnowledgePatch ||
			action == ActionKnowledgeDelete ||
			action == ActionClassesList ||
			action == ActionClassesGet ||
			action == ActionFilesUpload ||
			action == ActionFilesGet ||
			action == ActionFilesList
	case model.RoleSuperAdmin:
		return true
	default:
		return false
	}
}
