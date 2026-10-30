import { apiGet, apiDelete } from '../utils/request';
import type { FileInfo, FileURLResp, GetFileListReq, PageResult } from './types';
export const fetchFiles = (params: GetFileListReq) => apiGet<PageResult<FileInfo>>('/sys/file/list', params);
export const fetchFileURL = (id: number) => apiGet<FileURLResp>('/sys/file/' + id + '/url');
export const deleteFile = (id: number) => apiDelete('/sys/file/' + id);
