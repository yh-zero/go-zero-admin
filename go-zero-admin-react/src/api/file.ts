import { apiGet, apiDelete, apiPost } from '../utils/request';
import type { FileInfo, FileURLResp, GetFileListReq, PageResult, UploadFileReq, UploadFileResp } from './types';
export const fetchFiles = (params: GetFileListReq) => apiGet<PageResult<FileInfo>>('/sys/file/list', params);
export const fetchFileURL = (id: number) => apiGet<FileURLResp>('/sys/file/' + id + '/url');
export const deleteFile = (id: number) => apiDelete('/sys/file/' + id);
export const uploadFile = (values: UploadFileReq) => apiPost<UploadFileResp>('/sys/file/upload', values).then(result => result.file);
