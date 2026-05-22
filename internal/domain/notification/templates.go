package notification

import (
	"domainnest/internal/model"
	"fmt"
)

func PermissionGranted(node *model.DomainNode, level string) Notification {
	return Notification{
		Category:   CatPermissionGranted,
		Title:      "权限授予",
		Content:    fmt.Sprintf("你已被授予 %s 的 %s 权限", node.FullDomain, level),
		ActionType: "permission_grant",
		TargetType: "domain_node",
		TargetID:   node.ID,
		Priority:   PriorityInfo,
	}
}

func PermissionRevoked(node *model.DomainNode) Notification {
	return Notification{
		Category: CatPermissionRevoked,
		Title:    "权限撤销",
		Content:  fmt.Sprintf("你在 %s 的权限已被撤销", node.FullDomain),
		Priority: PriorityWarning,
	}
}

func PermissionRevokeRequest(node *model.DomainNode) Notification {
	return Notification{
		Category:   CatPermissionRevokeReq,
		Title:      "权限归还请求",
		Content:    fmt.Sprintf("管理员请求归还你在 %s 的权限", node.FullDomain),
		ActionType: "revoke_return",
		TargetType: "domain_node",
		TargetID:   node.ID,
		Priority:   PriorityWarning,
	}
}

func PermissionReturned(node *model.DomainNode) Notification {
	return Notification{
		Category: CatPermissionReturned,
		Title:    "权限已归还",
		Content:  fmt.Sprintf("你在 %s 的权限已归还", node.FullDomain),
		Priority: PriorityInfo,
	}
}

func DomainTransferredTo(node *model.DomainNode, fromUsername string) Notification {
	return Notification{
		Category:   CatDomainTransferred,
		Title:      "域名转让",
		Content:    fmt.Sprintf("%s 将 %s 转让给你", fromUsername, node.FullDomain),
		ActionType: "view_domain",
		TargetType: "domain_node",
		TargetID:   node.ID,
		Priority:   PriorityInfo,
	}
}

func DomainTransferredAway(node *model.DomainNode, toUsername string) Notification {
	return Notification{
		Category: CatDomainTransferred,
		Title:    "域名已转出",
		Content:  fmt.Sprintf("你已将 %s 转让给 %s", node.FullDomain, toUsername),
		Priority: PriorityInfo,
	}
}

func DomainTransferredWithDelegations(node *model.DomainNode, delegationCount int) Notification {
	return Notification{
		Category:   CatDomainTransferred,
		Title:      "域名转让 — 存在委派权限",
		Content:    fmt.Sprintf("域名 %s 有 %d 个委派权限，请查看授权管理决定是否保留", node.FullDomain, delegationCount),
		ActionType: "view_permissions",
		TargetType: "domain_node",
		TargetID:   node.ID,
		Priority:   PriorityWarning,
	}
}

func DomainReclaimed(node *model.DomainNode, byUsername string) Notification {
	return Notification{
		Category: CatDomainReclaimed,
		Title:    "域名被回收",
		Content:  fmt.Sprintf("%s 回收了你的域名 %s", byUsername, node.FullDomain),
		Priority: PriorityWarning,
	}
}

func DomainReactivated(node *model.DomainNode) Notification {
	return Notification{
		Category:   CatDomainReactivated,
		Title:      "域名已激活",
		Content:    fmt.Sprintf("域名 %s 已重新激活", node.FullDomain),
		ActionType: "view_domain",
		TargetType: "domain_node",
		TargetID:   node.ID,
		Priority:   PriorityInfo,
	}
}

func RolePromoted() Notification {
	return Notification{
		Category: CatAdminPromoted,
		Title:    "角色变更",
		Content:  "你已被提升为管理员",
		Priority: PriorityInfo,
	}
}

func RoleDemoted() Notification {
	return Notification{
		Category: CatAdminDemoted,
		Title:    "角色变更",
		Content:  "你已被降级为普通用户",
		Priority: PriorityWarning,
	}
}

func AdminPasswordReset() Notification {
	return Notification{
		Category: CatAdminPasswordReset,
		Title:    "密码已被重置",
		Content:  "管理员重置了你的密码",
		Priority: PriorityWarning,
	}
}

func AccountDisabled() Notification {
	return Notification{
		Category: CatAdminDisabled,
		Title:    "账号已禁用",
		Content:  "你的账号已被禁用",
		Priority: PriorityError,
	}
}

func InviteGranted(amount int) Notification {
	return Notification{
		Category: CatAdminGrantInvite,
		Title:    "邀请额度变更",
		Content:  fmt.Sprintf("你收到了 %d 个邀请额度", amount),
		Priority: PriorityInfo,
	}
}

func InviteRevoked(amount int) Notification {
	return Notification{
		Category: CatAdminRevokeInvite,
		Title:    "邀请额度变更",
		Content:  fmt.Sprintf("你的 %d 个邀请额度已被收回", amount),
		Priority: PriorityWarning,
	}
}

func InviteCodeUsed(username string) Notification {
	return Notification{
		Category: CatInviteCodeUsed,
		Title:    "邀请码被使用",
		Content:  fmt.Sprintf("%s 通过你的邀请码注册了账号", username),
		Priority: PriorityInfo,
	}
}
func Welcome() Notification {
	return Notification{
		Category: CatUserWelcome,
		Title:    "欢迎加入",
		Content:  "欢迎加入 DomainNest！",
		Priority: PriorityInfo,
	}
}

func EmailVerifiedNotification(email string) Notification {
	return Notification{
		Category: CatEmailVerified,
		Title:    "邮箱已验证",
		Content:  fmt.Sprintf("你的邮箱 %s 已验证成功", email),
		Priority: PriorityInfo,
	}
}

func FriendRequestReceived(senderUsername string, requestID uint64) Notification {
	return Notification{
		Category:   CatFriendRequest,
		Title:      "好友请求",
		Content:    fmt.Sprintf("%s 想添加你为好友", senderUsername),
		ActionType: "friend_request",
		TargetType: "friend_request",
		TargetID:   requestID,
		Priority:   PriorityInfo,
	}
}

func FriendRequestAccepted(username string) Notification {
	return Notification{
		Category: CatFriendAccepted,
		Title:    "好友请求通过",
		Content:  fmt.Sprintf("%s 接受了你的好友请求", username),
		Priority: PriorityInfo,
	}
}

func FriendRequestRejected(username string) Notification {
	return Notification{
		Category: CatFriendRejected,
		Title:    "好友请求被拒",
		Content:  fmt.Sprintf("%s 拒绝了你的好友请求", username),
		Priority: PriorityInfo,
	}
}

func DomainClaimed(node *model.DomainNode) Notification {
	return Notification{
		Category:   CatProviderClaimed,
		Title:      "域名认领",
		Content:    fmt.Sprintf("域名 %s 已认领成功", node.FullDomain),
		ActionType: "view_domain",
		TargetType: "domain_node",
		TargetID:   node.ID,
		Priority:   PriorityInfo,
	}
}

func DomainReclaimedByProvider(node *model.DomainNode, byUsername string) Notification {
	return Notification{
		Category: CatProviderReclaimed,
		Title:    "域名被回收",
		Content:  fmt.Sprintf("%s 通过服务商回收了域名 %s", byUsername, node.FullDomain),
		Priority: PriorityWarning,
	}
}

func SyncFailedNotification(domain string, recordID uint64, detail string) Notification {
	return Notification{
		Category:   CatSyncFailed,
		Title:      "同步失败",
		Content:    fmt.Sprintf("DNS 记录同步失败：%s", detail),
		ActionType: "retry_sync",
		TargetType: "dns_record",
		TargetID:   recordID,
		Priority:   PriorityError,
	}
}

func DomainDeleted(node *model.DomainNode) Notification {
	return Notification{
		Category:   CatDomainDeleted,
		Title:      "域名已删除",
		Content:    fmt.Sprintf("域名 %s 已被删除", node.FullDomain),
		ActionType: "view_domain",
		TargetType: "domain_node",
		TargetID:   node.ID,
		Priority:   PriorityWarning,
	}
}

func DomainArchived(node *model.DomainNode, byUsername string) Notification {
	return Notification{
		Category:   CatDomainArchived,
		Title:      "域名已归档",
		Content:    fmt.Sprintf("域名 %s 已被 %s 归档", node.FullDomain, byUsername),
		ActionType: "view_archived",
		TargetType: "domain_node",
		TargetID:   node.ID,
		Priority:   PriorityWarning,
	}
}

func DomainRestored(node *model.DomainNode) Notification {
	return Notification{
		Category:   CatDomainReactivated,
		Title:      "域名已恢复",
		Content:    fmt.Sprintf("域名 %s 已从归档恢复", node.FullDomain),
		ActionType: "view_domain",
		TargetType: "domain_node",
		TargetID:   node.ID,
		Priority:   PriorityInfo,
	}
}

func SubdomainReturned(node *model.DomainNode, byUsername string) Notification {
	return Notification{
		Category:   CatDomainTransferred,
		Title:      "子域名已归还",
		Content:    fmt.Sprintf("%s 归还了子域名 %s", byUsername, node.FullDomain),
		ActionType: "view_domain",
		TargetType: "domain_node",
		TargetID:   node.ID,
		Priority:   PriorityInfo,
	}
}

func RecordTrashed(record *model.DNSRecord, domain string) Notification {
	return Notification{
		Category:   CatRecordTrashed,
		Title:      "记录已移入回收站",
		Content:    fmt.Sprintf("DNS 记录 %s.%s (%s) 已移入回收站", record.Host, domain, record.RecordType),
		ActionType: "view_trash",
		TargetType: "dns_record",
		TargetID:   record.ID,
		Priority:   PriorityInfo,
	}
}

func RecordRestored(record *model.DNSRecord, domain string) Notification {
	return Notification{
		Category:   CatRecordRestored,
		Title:      "记录已恢复",
		Content:    fmt.Sprintf("DNS 记录 %s.%s (%s) 已从回收站恢复", record.Host, domain, record.RecordType),
		ActionType: "view_record",
		TargetType: "dns_record",
		TargetID:   record.ID,
		Priority:   PriorityInfo,
	}
}

func ProviderDeleted(providerName string) Notification {
	return Notification{
		Category: CatProviderDeleted,
		Title:    "服务商已删除",
		Content:  fmt.Sprintf("DNS 服务商 %s 已被删除", providerName),
		Priority: PriorityInfo,
	}
}

func DDNSUpdateOK(domain string, recordID uint64) Notification {
	return Notification{
		Category:   CatDDNSUpdateOK,
		Title:      "DDNS 更新成功",
		Content:    fmt.Sprintf("DDNS 记录 %s 更新成功", domain),
		ActionType: "view_record",
		TargetType: "dns_record",
		TargetID:   recordID,
		Priority:   PriorityInfo,
	}
}

func DDNSUpdateFailed(domain string, recordID uint64, detail string) Notification {
	return Notification{
		Category:   CatDDNSUpdateFailed,
		Title:      "DDNS 更新失败",
		Content:    fmt.Sprintf("DDNS 记录 %s 更新失败：%s", domain, detail),
		ActionType: "view_record",
		TargetType: "dns_record",
		TargetID:   recordID,
		Priority:   PriorityError,
	}
}
