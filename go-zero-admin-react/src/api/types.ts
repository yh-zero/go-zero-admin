// 与后端约定的统一类型（snake_case 对齐 REST 返回）
export interface ApiResp<T = unknown> {
  code: number;
  msg: string;
  result?: T;
}

export interface LoginReq {
  username: string;
  password: string;
  captcha_id: string;
  captcha: string;
}

export interface LoginResp {
  access_token: string;
  expires_at: number;
}

export interface CaptchaResp {
  captcha_id: string;
  image_base64: string;
  expires_at: number;
}

export interface MenuInfo {
  id: number;
  parent_id: number;
  title: string;
  name: string;
  type: number; // 1目录 2菜单 3按钮
  path: string;
  component: string;
  perm_code: string;
  icon: string;
  sort: number;
  visible: boolean;
  status: number;
  children?: MenuInfo[];
}
