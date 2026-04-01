package authz

import "manage/internal/model"

func Authorize(role int, action string) bool {
	switch role {
	case model.RoleStudent:
		return action == ActionGetMe ||
			action == ActionKnowledgeSearch ||
			action == ActionFilesUpload ||
			action == ActionFilesGet ||
			action == ActionFilesList
	case model.RoleCadre:
		return action == ActionUsersList ||
			action == ActionUsersGet ||
			action == ActionKnowledgeSearch ||
			action == ActionKnowledgeList ||
			action == ActionKnowledgeGet ||
			action == ActionKnowledgeCreate ||
			action == ActionKnowledgePatch ||
			action == ActionKnowledgeDelete ||
			action == ActionClassesList ||
			action == ActionClassesGet ||
			action == ActionFilesUpload ||
			action == ActionFilesGet ||
			action == ActionFilesList ||
			action == ActionFilesDelete
	case model.RoleTeacher:
		return action == ActionUsersList ||
			action == ActionUsersGet ||
			action == ActionKnowledgeSearch ||
			action == ActionKnowledgeList ||
			action == ActionKnowledgeGet ||
			action == ActionKnowledgeCreate ||
			action == ActionKnowledgePatch ||
			action == ActionKnowledgeDelete ||
			action == ActionClassesList ||
			action == ActionClassesGet ||
			action == ActionFilesUpload ||
			action == ActionFilesGet ||
			action == ActionFilesList ||
			action == ActionFilesDelete
	case model.RoleSuperAdmin:
		return true
	default:
		return false
	}
}
