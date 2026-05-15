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
			action == ActionPartyflowMeGet ||
			action == ActionAnnouncementsList ||
			action == ActionAnnouncementsGet ||
			action == ActionApprovalsCreate ||
			action == ActionApprovalsMyList ||
			action == ActionApprovalsGet ||
			action == ActionApprovalsWithdraw ||
			action == ActionCertificatesMyList ||
			action == ActionCertificatesGet
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
			action == ActionPartyflowMeGet ||
			action == ActionPartyflowStatusesList ||
			action == ActionPartyflowStatusesGet ||
			action == ActionPartyflowStatusesCreate ||
			action == ActionPartyflowStatusesPatch ||
			action == ActionPartyflowStatusesImport ||
			action == ActionPartyflowEventsCreate ||
			action == ActionPartyflowRulesList ||
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
			action == ActionApprovalsRemind ||
			action == ActionCertificatesMyList ||
			action == ActionCertificatesGet
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
			action == ActionPartyflowMeGet ||
			action == ActionPartyflowStatusesList ||
			action == ActionPartyflowStatusesGet ||
			action == ActionPartyflowStatusesCreate ||
			action == ActionPartyflowStatusesPatch ||
			action == ActionPartyflowStatusesImport ||
			action == ActionPartyflowEventsCreate ||
			action == ActionPartyflowRulesList ||
			action == ActionPartyflowRulesPatch ||
			action == ActionPartyflowRemindersScan ||
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
			action == ActionApprovalsOverdueScan ||
			action == ActionCertificatesMyList ||
			action == ActionCertificatesGet ||
			action == ActionCertificatesTemplateAdminList ||
			action == ActionCertificatesTemplateToggle ||
			action == ActionCertificatesApplicationRegenerate ||
			action == ActionCertificatesCertificateRegenerate ||
			action == ActionCertificatesRevoke
	case model.RoleSuperAdmin:
		return true
	default:
		return false
	}
}
