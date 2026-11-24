import { apiGet, apiDelete, apiPost, apiPostForm } from '../utils/request';
import type { FileDetailResp, FileInfo, FileStatsResp, FileURLResp, GetFileListReq, PageResult, UploadFileReq, UploadFileResp } from './types';
export const fetchFiles = (params: GetFileListReq) => apiGet<PageResult<FileInfo>>('/sys/file/list', params);
export const fetchFileURL = (id: number) => apiGet<FileURLResp>('/sys/file/' + id + '/url');
export const deleteFile = (id: number) => apiDelete('/sys/file/' + id);
export const uploadFile = (values: UploadFileReq) => apiPost<UploadFileResp>('/sys/file/upload', values, { timeout: 50_000 }).then(result => result.file);
export const uploadMultipart = (file: File, scene: UploadFileReq['scene'], visibility: UploadFileReq['visibility']) => {
  const data = new FormData();
  data.append('file', file);
  data.append('scene', scene);
  data.append('visibility', visibility);
  return apiPostForm<UploadFileResp>('/sys/file/upload-multipart', data, { timeout: 50_000 }).then(result => result.file);
};
export const fetchFileDetail = (id: number) => apiGet<FileDetailResp>('/sys/file/' + id);
export const fetchFileStats = () => apiGet<FileStatsResp>('/sys/file/stats');
