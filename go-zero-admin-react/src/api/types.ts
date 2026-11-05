// REST 字段以共享契约和已核对的后端生成类型为准。
export interface ApiResp<T = unknown> { code: number; msg: string; result?: T }
export interface PageParams { page?: number; page_size?: number }
export interface PageResult<T> { list: T[]; total: number }
export interface ListResult<T> { list: T[] }
export interface LoginReq { username: string; password: string; captcha_id?: string; captcha?: string }
export interface LoginResp {
  access_token: string; expires_at: number; refresh_token: string; refresh_expires_at: number; session_id: string;
}
export interface DeviceSession {
  id: string; device_name: string; ip: string; user_agent: string;
  created_at: number; last_seen_at: number; expires_at: number; is_current: boolean;
}
export interface CaptchaResp { enabled: boolean; captcha_id: string; image_base64: string; expires_at: number }
export interface MenuInfo {
  id: number; parent_id: number; title: string; name: string; type: number;
  path: string; component: string; perm_code: string; icon: string;
  sort: number; visible: boolean; status: number; created_at: number; children?: MenuInfo[];
}
export interface CreateMenuReq {
  parent_id?: number; title: string; name?: string; type: number; path?: string;
  component?: string; perm_code?: string; icon?: string; sort?: number; visible?: boolean; status?: number;
}
export type UpdateMenuReq = Partial<{ [K in keyof CreateMenuReq]: CreateMenuReq[K] | null }>;
export interface GetMenuListReq extends PageParams { parent_id?: number; type?: number }
export interface GetMenuTreeReq { type?: number; status?: number }
export interface UserInfo {
  id: number; username: string; nickname: string; real_name: string; email: string; phone: string;
  dept_id: number; dept_name: string; avatar_file_id: number; avatar_url: string; gender: number; status: number;
  remark: string; last_login_at: number; last_login_ip: string; created_at: number; updated_at: number;
  role_ids: number[];
}
export interface UpdateUserReq {
  nickname?: string; real_name?: string; email?: string; phone?: string; dept_id?: number;
  avatar_file_id?: number; gender?: number; status?: number; remark?: string; role_ids?: number[];
}
export interface CreateUserReq extends UpdateUserReq { username: string; password: string }
export interface GetUserListReq extends PageParams { username?: string; nickname?: string; dept_id?: number; status?: number }
export interface ChangePasswordReq { old_password: string; new_password: string }
export interface DeptInfo {
  id: number; parent_id: number; name: string; leader: string; leader_user_id: number;
  member_count: number; members_visible: boolean; phone: string; email: string;
  sort: number; status: number; path: string; created_at: number;
}
export interface CreateDeptReq {
  parent_id?: number; name: string; leader?: string; leader_user_id?: number;
  phone?: string; email?: string; sort?: number; status?: number;
}
export type UpdateDeptReq = Partial<CreateDeptReq>;
export interface GetDeptTreeReq { status?: number }
export interface RoleInfo {
  id: number; name: string; code: string; description: string; data_scope: string; sort: number;
  status: number; menu_ids: number[]; user_count: number; created_at: number; updated_at: number;
}
export interface CreateRoleReq {
  name: string; code: string; description?: string; data_scope?: string; sort?: number; status?: number; menu_ids?: number[];
}
export type UpdateRoleReq = Partial<{ [K in keyof CreateRoleReq]: CreateRoleReq[K] | null }>;
export interface GetRoleListReq extends PageParams { name?: string; code?: string; status?: number }
export interface DictItem { label: string; value: string; sort: number }
export interface DictInfo {
  id: number; name: string; code: string; items: DictItem[]; description: string;
  status: number; created_at: number; updated_at: number;
}
export interface CreateDictReq { name: string; code: string; items?: DictItem[]; description?: string; status?: number }
export type UpdateDictReq = Partial<{ [K in keyof CreateDictReq]: CreateDictReq[K] | null }>;
export interface GetDictListReq extends PageParams { name?: string; code?: string; status?: number }
export interface ApiInfo { id: number; group: string; method: string; path: string; title: string; created_at: number }
export interface GetApiListReq extends PageParams { group?: string; method?: string; path?: string }
export interface SyncApiResp { added: number; updated: number; removed: number }
export interface FileInfo {
  id: number; bucket: string; object_key: string; original_name: string; ext: string; mime: string; size: number;
  visibility: string; ref_count: number; uploader_id: number; url?: string; created_at: number;
}
export interface GetFileListReq extends PageParams { visibility?: string; uploader_id?: number }
export interface FileURLResp { url: string; expires_at: number }
export interface UploadFileReq {
  name: string; content_base64: string; scene: 'common' | 'avatar' | 'doc'; visibility: 'public' | 'private';
}
export interface UploadFileResp { file: FileInfo }
export interface AuditLogInfo {
  id: number; user_id: number; username: string; module: string; action: string; method: string;
  path: string; result: number; error_msg: string; ip: string; duration_ms: number; created_at: number;
}
export interface GetAuditLogListReq extends PageParams { module?: string; action?: string; user_id?: number; result?: number }
export interface ProfileInfo {
  id: number; username: string; nickname: string; real_name: string; email: string;
  phone: string; avatar_url: string; avatar_file_id: number; gender: number; dept_name: string; is_super_admin: boolean;
}
export interface UpdateProfileReq {
  nickname?: string; real_name?: string; email?: string; phone?: string; avatar_file_id?: number; gender?: number;
}
export type AppUserStatus = 'pending' | 'active' | 'disabled' | 'rejected';
export interface AppUserInfo {
  id: number; account: string; identity_kind: string; nickname: string;
  avatar_file_id: number; avatar_url: string; status: AppUserStatus; review_reason: string;
  email_verified: boolean; phone_verified: boolean; last_login_at: number; created_at: number;
}
export interface AppAdminUserInfo extends AppUserInfo { remark: string }
export interface UpdateAppUserReq { nickname?: string; remark?: string }
export interface AppMembershipPlan { label: string; value: string }
export interface GetAppUserListReq extends PageParams { account?: string; status?: AppUserStatus }
export interface ReviewAppUserReq { status: AppUserStatus; reason?: string }
export interface AppMembershipInfo {
  id: number; user_id: number; plan_code: string; benefits: string; request_id: string;
  granted_by: number; granted_by_admin: boolean; starts_at: number; expires_at: number;
  revoked: boolean; revoked_at: number; created_at: number;
}
export interface GetAppMembershipListReq extends PageParams {
  user_id?: number; plan_code?: string; active_only?: boolean;
}
export interface GrantAppMembershipReq {
  user_id: number; plan_code: string; benefits?: string; request_id: string; duration_days: number;
}
export type AppFeedbackStatus = 'pending' | 'resolved';
export interface AppFeedbackInfo {
  id: number; user_id: number; account: string; nickname: string; content: string; reply: string;
  status: AppFeedbackStatus; created_at: number; updated_at: number;
}
export interface GetAppFeedbackListReq extends PageParams { user_id?: number; status?: AppFeedbackStatus }
export interface UpdateAppFeedbackReq { reply?: string; status?: AppFeedbackStatus }

// ===== 租户管理（仅平台栈） =====
export interface TenantInfo {
  id: number;
  slug: string;
  domain: string;
  compose_project: string;
  status: string; // creating | active | stopped | error
  error_msg: string;
  admin_account_ready: boolean;
  created_by: number;
  created_at: number;
  updated_at: number;
}
export interface GetTenantListReq extends PageParams { status?: string }
export interface GetTenantResp { tenant: TenantInfo }
export interface CreateTenantReq {
  slug: string;
  admin_username?: string;
  admin_password?: string;
}
export interface CreateTenantResp {
  tenant: TenantInfo;
  admin_password: string; // 仅此一次返回
}

// ===== AI 会话与工作流 =====
export type AiMessageStatus = 'running' | 'succeeded' | 'failed';
export type AiTaskStatus = 'pending' | 'running' | 'pending_approval' | 'succeeded' | 'failed' | 'rejected';
export interface AiConversationInfo {
  id: number; title: string; user_id: number; message_count: number; created_at: number; updated_at: number;
}
export interface AiChatMessage {
  id: number; role: string; content: string; tool_calls: string; tokens: number; created_at: number;
  status: AiMessageStatus; error: string;
}
export interface AiWorkflowInfo {
  id: number; name: string; definition: string; interval_seconds: number; enabled: number;
  next_run_at: number; created_by: number; created_at: number; updated_at: number;
}
export interface AiTaskInfo {
  id: number; workflow_id: number; conversation_id: number; status: AiTaskStatus; input: string;
  result: string; error: string; approved_by: number; created_at: number; completed_at: number; next_step: number;
}
export interface AiToolInfo { name: string; description: string; parameters_schema: string }
export interface AiConversationResp { conversation: AiConversationInfo }
export interface AiGetConversationReq { before_id?: number; page_size?: number }
export interface AiGetConversationResp extends AiConversationResp {
  messages: AiChatMessage[]; has_more: boolean; next_before_id: number;
}
// 同一逻辑提交重试必须复用 request_id；用户有意发起新调用时由调用者生成新的 UUID。
export interface AiSendMessageReq { request_id: string; conversation_id: number; content: string }
export interface AiSendMessageResult {
  conversation_id: number; conversation: AiConversationInfo | null;
  user_message: AiChatMessage; ai_message: AiChatMessage;
}
export interface AiStreamMeta { conversation_id: number; user_message_id: number; ai_message_id: number }
export interface AiStreamDone { tokens: number; usage_complete: boolean; result: AiSendMessageResult }
export interface AiCreateWorkflowReq { name: string; definition: string; interval_seconds?: number }
export interface AiUpdateWorkflowReq { name?: string; definition?: string; interval_seconds?: number; enabled?: 0 | 1 }
export interface AiWorkflowResp { workflow: AiWorkflowInfo }
export interface AiRunWorkflowResp { task_id: number; status: AiTaskStatus }
export interface AiListTasksReq extends PageParams { status?: AiTaskStatus; workflow_id?: number }
export interface AiTaskResp { task: AiTaskInfo }
export interface AiApproveTaskReq { expected_step: number }
export interface AiRejectTaskReq extends AiApproveTaskReq { reason: string }
