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
			action == ActionFilesList ||
			action == ActionAnnouncementsList ||
			action == ActionAnnouncementsGet ||
			action == ActionApprovalsCreate ||
			action == ActionApprovalsMyList ||
			action == ActionApprovalsGet ||
			action == ActionApprovalsWithdraw
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
			action == ActionFilesList ||
			action == ActionAnnouncementsList ||
			action == ActionAnnouncementsGet ||
			action == ActionAnnouncementsAdminList ||
			action == ActionAnnouncementsAdminGet ||
			action == ActionAnnouncementsCreate ||
			action == ActionAnnouncementsPatch ||
			action == ActionAnnouncementsPublish ||
			action == ActionAnnouncementsArchive ||
			action == ActionApprovalsList ||
			action == ActionApprovalsGet ||
			action == ActionApprovalsRemind
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
			action == ActionFilesList ||
			action == ActionAnnouncementsList ||
			action == ActionAnnouncementsListAll ||
			action == ActionAnnouncementsGet ||
			action == ActionAnnouncementsGetAll ||
			action == ActionAnnouncementsAdminList ||
			action == ActionAnnouncementsAdminGet ||
			action == ActionAnnouncementsCreate ||
			action == ActionAnnouncementsPatch ||
			action == ActionAnnouncementsPublish ||
			action == ActionAnnouncementsArchive ||
			action == ActionApprovalsList ||
			action == ActionApprovalsGet ||
			action == ActionApprovalsReview ||
			action == ActionApprovalsAssign ||
			action == ActionApprovalsRemind ||
			action == ActionApprovalsOverdueScan
	case model.RoleSuperAdmin:
		return true
	default:
		return false
	}
}
