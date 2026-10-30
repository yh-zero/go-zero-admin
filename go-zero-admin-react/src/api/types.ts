// REST 字段以共享契约和已核对的后端生成类型为准。
export interface ApiResp<T = unknown> { code: number; msg: string; result?: T }
export interface PageParams { page?: number; page_size?: number }
export interface PageResult<T> { list: T[]; total: number }
export interface ListResult<T> { list: T[] }
export interface LoginReq { username: string; password: string; captcha_id: string; captcha: string }
export interface LoginResp { access_token: string; expires_at: number }
export interface CaptchaResp { captcha_id: string; image_base64: string; expires_at: number }
export interface MenuInfo {
  id: number; parent_id: number; title: string; name: string; type: number;
  path: string; component: string; perm_code: string; icon: string;
  sort: number; visible: boolean; status: number; created_at: number; children?: MenuInfo[];
}
export interface CreateMenuReq {
  parent_id?: number; title: string; name?: string; type: number; path?: string;
  component?: string; perm_code?: string; icon?: string; sort?: number; visible?: boolean; status?: number;
}
export type UpdateMenuReq = CreateMenuReq;
export interface GetMenuListReq extends PageParams { parent_id?: number; type?: number }
export interface GetMenuTreeReq { type?: number; status?: number }
export interface UserInfo {
  id: number; username: string; nickname: string; real_name: string; email: string; phone: string;
  dept_id: number; dept_name: string; avatar_file_id: number; gender: number; status: number;
  remark: string; last_login_at: number; last_login_ip: string; created_at: number; updated_at: number;
}
export interface UpdateUserReq {
  nickname?: string; real_name?: string; email?: string; phone?: string; dept_id?: number;
  avatar_file_id?: number; gender?: number; status?: number; remark?: string;
}
export interface CreateUserReq extends UpdateUserReq { username: string; password: string }
export interface GetUserListReq extends PageParams { username?: string; nickname?: string; dept_id?: number; status?: number }
export interface ChangePasswordReq { old_password: string; new_password: string }
export interface DeptInfo {
  id: number; parent_id: number; name: string; leader: string; phone: string; email: string;
  sort: number; status: number; path: string; created_at: number;
}
export interface CreateDeptReq {
  parent_id?: number; name: string; leader?: string; phone?: string; email?: string; sort?: number; status?: number;
}
export type UpdateDeptReq = CreateDeptReq;
export interface GetDeptTreeReq { status?: number }
export interface RoleInfo {
  id: number; name: string; code: string; description: string; data_scope: string; sort: number;
  status: number; menu_ids: number[]; user_count: number; created_at: number; updated_at: number;
}
export interface CreateRoleReq {
  name: string; code: string; description?: string; data_scope?: string; sort?: number; status?: number; menu_ids?: number[];
}
export type UpdateRoleReq = CreateRoleReq;
export interface GetRoleListReq extends PageParams { name?: string; code?: string; status?: number }
export interface DictItem { label: string; value: string; sort: number }
export interface DictInfo {
  id: number; name: string; code: string; items: DictItem[]; description: string;
  status: number; created_at: number; updated_at: number;
}
export interface CreateDictReq { name: string; code: string; items?: DictItem[]; description?: string; status?: number }
export type UpdateDictReq = CreateDictReq;
export interface GetDictListReq extends PageParams { name?: string; code?: string; status?: number }
export interface ApiInfo { id: number; group: string; method: string; path: string; title: string; created_at: number }
export interface GetApiListReq extends PageParams { group?: string; method?: string; path?: string }
export interface FileInfo {
  id: number; bucket: string; object_key: string; original_name: string; ext: string; mime: string; size: number;
  visibility: string; ref_count: number; uploader_id: number; url?: string; created_at: number;
}
export interface GetFileListReq extends PageParams { visibility?: string; uploader_id?: number }
export interface FileURLResp { url: string; expires_at: number }
export interface AuditLogInfo {
  id: number; user_id: number; username: string; module: string; action: string; method: string;
  path: string; result: number; error_msg: string; ip: string; duration_ms: number; created_at: number;
}
export interface GetAuditLogListReq extends PageParams { module?: string; action?: string; user_id?: number; result?: number }
export interface ProfileInfo {
  id: number; username: string; nickname: string; real_name: string; email: string;
  phone: string; avatar_url: string; gender: number; dept_name: string;
}
export interface UpdateProfileReq {
  nickname?: string; real_name?: string; email?: string; phone?: string; avatar_url?: string; gender?: number;
}
