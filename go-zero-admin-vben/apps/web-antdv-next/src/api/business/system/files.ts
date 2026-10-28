import type { FileVisibility } from '../base';
import type { PageQuery, PageResult } from './types';

import { requestClient } from '#/api/request';

export interface FileResource {
  id: number;
  name: string;
  mime: string;
  size: number;
  ownerId: number;
  departmentId: number;
  visibility: FileVisibility;
  status: 'active' | 'deleted' | 'deleting';
  references: number;
  createdAt: string;
}
export interface FileQuery extends PageQuery {
  name?: string;
  status?: FileResource['status'];
}
export interface FileURL {
  url: string;
  expiresIn: number;
}
export interface FileReference {
  fileId: number;
  objectType: string;
  objectId: string;
}
export const getFiles = (params: FileQuery) =>
  requestClient.get<PageResult<FileResource>>('/v1/sys/files/list', { params });
export const getFileURL = (id: number) =>
  requestClient.get<FileURL>('/v1/sys/files/url', { params: { id } });
export const deleteFile = (id: number) =>
  requestClient.delete('/v1/sys/files/resource', { data: { id } });
export const addFileReference = (data: FileReference) =>
  requestClient.post('/v1/sys/files/reference', data);
export const removeFileReference = (data: FileReference) =>
  requestClient.delete('/v1/sys/files/reference', { data });
