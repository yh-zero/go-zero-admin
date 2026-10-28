export interface BackendUser {
  ID: number;
  authority?: { authorityName?: string; defaultRouter?: string };
  email?: string;
  phone?: string;
  enable?: number;
  authorityId: number;
  headerImg?: string;
  nickName?: string;
  userName: string;
}

export interface LoginResult {
  accessExpire: number;
  accessToken: string;
  userInfo: BackendUser;
}

export interface BackendMenu {
  btns?: null | Record<string, number>;
  children?: BackendMenu[] | null;
  component?: string;
  hidden?: boolean;
  meta?: { icon?: string; keepAlive?: boolean; title?: string };
  name: string;
  path: string;
  sort?: number;
}
