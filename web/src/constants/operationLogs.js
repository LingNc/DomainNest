export const actionGroups = [
  {
    label: '域名',
    options: [
      { value: 'create_domain', label: '创建域名' },
      { value: 'transfer_domain', label: '转让域名' },
      { value: 'delete_domain', label: '删除域名' },
      { value: 'create_root_domain', label: '创建根域名' },
      { value: 'assign_domain', label: '分配域名' },
    ],
  },
  {
    label: '记录',
    options: [
      { value: 'create_record', label: '创建记录' },
      { value: 'update_record', label: '更新记录' },
      { value: 'delete_record', label: '删除记录' },
      { value: 'toggle_record', label: '切换记录' },
      { value: 'batch_delete', label: '批量删除' },
      { value: 'batch_toggle', label: '批量切换' },
      { value: 'import', label: '导入' },
    ],
  },
  {
    label: '权限',
    options: [
      { value: 'grant_permission', label: '授权' },
      { value: 'revoke_permission', label: '撤销权限' },
      { value: 'permission_granted', label: '被授予权限' },
      { value: 'permission_revoked', label: '权限被回收' },
      { value: 'revoke_request', label: '回收请求' },
      { value: 'accept_return', label: '接受回收' },
      { value: 'reject_return', label: '拒绝回收' },
      { value: 'assign_pending_records', label: '分配待处理记录' },
      { value: 'delete_pending_records', label: '删除待处理记录' },
    ],
  },
  {
    label: '用户',
    options: [
      { value: 'register', label: '注册' },
      { value: 'login', label: '登录' },
      { value: 'update_profile', label: '更新资料' },
      { value: 'change_password', label: '修改密码' },
      { value: 'reset_token', label: '重置令牌' },
      { value: 'upload_avatar', label: '上传头像' },
      { value: 'delete_account', label: '注销账号' },
      { value: 'update_user', label: '更新用户' },
      { value: 'admin_reset_password', label: '管理员重置密码' },
      { value: 'disable_user', label: '禁用用户' },
      { value: 'promote_to_admin', label: '提升为管理员' },
      { value: 'demote_from_admin', label: '降级为普通用户' },
    ],
  },
  {
    label: '邀请',
    options: [
      { value: 'grant_invite', label: '分配邀请' },
      { value: 'revoke_invite', label: '收回邀请' },
      { value: 'invite_granted', label: '获赠邀请额度' },
      { value: 'invite_revoked', label: '邀请额度被回收' },
      { value: 'admin_grant', label: '管理员分配' },
    ],
  },
  {
    label: '好友',
    options: [
      { value: 'send_friend_request', label: '发送好友请求' },
      { value: 'accept_friend', label: '接受好友' },
      { value: 'reject_friend', label: '拒绝好友' },
      { value: 'remove_friend', label: '删除好友' },
    ],
  },
  {
    label: '提供商',
    options: [
      { value: 'create_provider', label: '创建提供商' },
      { value: 'update_provider', label: '更新提供商' },
      { value: 'delete_provider', label: '删除提供商' },
      { value: 'claim_domain', label: '认领域名' },
    ],
  },
  {
    label: '设置',
    options: [
      { value: 'update_settings', label: '更新设置' },
      { value: 'smtp_test', label: 'SMTP 测试' },
    ],
  },
]

export const targetTypeOptions = [
  { value: 'domain_node', label: '域名节点' },
  { value: 'dns_record', label: 'DNS 记录' },
  { value: 'user', label: '用户' },
  { value: 'setting', label: '设置' },
  { value: 'provider', label: '提供商' },
  { value: 'friend', label: '好友' },
]

export const actionLabelMap = Object.fromEntries(
  actionGroups.flatMap((g) => g.options.map((o) => [o.value, o.label]))
)
