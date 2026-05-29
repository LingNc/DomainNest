package errs

import "net/http"

// HTTPStatus returns the HTTP status code for a given error code.
// Defaults to 400 Bad Request for business errors.
func HTTPStatus(code Code) int {
	switch code {
	// Auth errors -> 401
	case InvalidCredentials, LoginRequired, AccountDisabled:
		return http.StatusUnauthorized

	// Permission errors -> 403
	case NoPermission, NoAccess, NoPermissionForRecord, NoPermissionForProvider,
		CannotGrantHigherPermission, OnlyHolderOrAdminAccept, OnlyHolderReject,
		CannotEditSuperAdmin, CannotEditOtherAdmin, OnlySuperAdminChangeRole,
		CannotResetSuperAdminPassword, CannotDisableSelf, OnlySuperAdminDisableSuperAdmin,
		CannotDisableOtherAdmin, OnlySuperAdminPromote, OnlySuperAdminDemote,
		CannotDemoteSuperAdmin, SuperAdminCannotDelete, CannotRevokeOwnerPermission,
		NotRequestReceiver, NotSubdomainOwner, NotSubdomainOwner2:
		return http.StatusForbidden

	// Not found errors -> 404
	case NodeNotFound, DomainNodeNotFound, RecordNotFound, DomainNotFound,
		UserNotFound, TargetUserNotFound, DNSProviderNotFound, ProviderNotFound,
		NotificationNotFound, RequestNotFound, FriendRelationNotFound,
		PermissionNotFound, PermissionRecordNotFound, InviteCodeNotFound,
		TokenNotFound, PresetNotFound, ParentNodeNotFound, ReceiverNotFound,
		InviterNotFound, InviteeNotFound, DomainNotFoundOrNoAccess, RecordNotInTrash:
		return http.StatusNotFound

	// Conflict errors -> 409
	case UsernameExists, UsernameTaken, DomainAlreadyExists, DomainAlreadyInSystem,
		RecordAlreadyExists, SubdomainInUse, AlreadyFriends, FriendRequestPending,
		RequestAlreadyProcessed, CannotMessageSelf, CannotAddSelfAsFriend,
		CannotAssignInviteToSelf, CannotRevokeSelfInvite, NoRevocableInviteQuota,
		NoRevocableInviteQuotaGlobal, CannotDeleteUsedInviteCode, UserAlreadyAdmin,
		UserNotAdmin, CannotReclaimOwnNode, NotificationActionNotSupported,
		NotificationAlreadyProcessed, DomainArchived, DomainNotArchived,
		NodeNotArchived, NodeNoProviderBound, NodeNotFromRecord, RootNodeCannotDowngrade,
		CannotDowngradeWithChildren, CannotDowngradeWithPermissions, CannotDowngradeWithProvider,
		ArchiveRootBeforeDelete, CannotDeleteWithChildren, NoRecordsForConversion,
		CannotReturnRootDomain, DomainDeletedCannotSync, DomainArchivedCannotSync,
		DomainNoProviderBound, ProviderDeletedCannotRestore:
		return http.StatusConflict

	// Internal errors -> 500
	case InternalError, GenerateTokenFailed, ResetTokenFailed, GenerateVerifyCodeFailed,
		CreateResetCodeFailed, ResetPasswordFailed, FileUploadFailed, ImageEncodeFailed,
		AvatarSaveFailed, UpdateEmailFailed, TakeoverFailed, GetTransferDomainsFailed,
		PasswordEncryptFailed:
		return http.StatusInternalServerError

	// Validation errors -> 400 (default)
	default:
		return http.StatusBadRequest
	}
}