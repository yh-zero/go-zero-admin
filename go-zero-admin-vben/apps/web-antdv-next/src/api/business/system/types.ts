export interface PageResult<T> {
  list: null | T[];
  total: number;
  page: number;
  pageSize: number;
}
export interface PageQuery {
  pageNo?: number;
  pageSize?: number;
  keyword?: string;
}
export interface Authority {
  authorityId: number;
  authorityName: string;
  parentId: number;
  defaultRouter?: string;
  children?: Authority[];
}
export interface User {
  ID: number;
  userName: string;
  nickName: string;
  headerImg: string;
  phone: string;
  email: string;
  enable: number;
  authorityId: number;
  authorities?: Authority[];
}
export interface MenuButton {
  ID?: number;
  name: string;
  desc: string;
  sysBaseMenuID?: number;
  permissionKey?: string;
}
export interface Menu {
  movePreviewVersion?: string;
  ID: number;
  parentId: number;
  path: string;
  name: string;
  component: string;
  sort: number;
  hidden: boolean;
  meta: {
    title: string;
    icon: string;
    keepAlive?: boolean;
    closeTab?: boolean;
    [key: string]: unknown;
  };
  children?: Menu[];
  parameters?: Array<Record<string, unknown>>;
  menuBtn?: MenuButton[];
}
export interface ApiResource {
  ID: number;
  path: string;
  method: string;
  apiGroup: string;
  description: string;
}
export interface Policy {
  path: string;
  method: string;
}
export interface Dictionary {
  ID: number;
  name: string;
  type: string;
  status: number;
  desc: string;
  sysDictionaryInfoList?: DictionaryItem[];
}
export interface DictionaryItem {
  ID: number;
  label: string;
  value: number;
  extend: string;
  status: number;
  sort: number;
  sysDictionaryID: number;
}
