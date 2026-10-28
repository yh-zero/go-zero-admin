import type {
  Dictionary,
  DictionaryItem,
  PageQuery,
  PageResult,
} from './types';

import { requestClient } from '#/api/request';
const prefix = '/v1/sys/dictionary/';
export const getDictionaries = () =>
  requestClient.get<{ list: Dictionary[] | null }>(
    `${prefix}getSysDictionaryList`,
  );
export const getDictionary = (params: {
  id?: number;
  type?: string;
  status?: number;
}) =>
  requestClient.get<Dictionary>(`${prefix}getSysDictionaryDetails`, { params });
export const createDictionary = (data: Dictionary) =>
  requestClient.post(`${prefix}createSysDictionary`, data);
export const updateDictionary = (data: Dictionary) =>
  requestClient.put(`${prefix}updateSysDictionary`, data);
export const deleteDictionary = (id: number) =>
  requestClient.delete(`${prefix}deleteSysDictionary`, { data: { id } });
export const getDictionaryItems = (
  params: PageQuery & {
    sysDictionaryID: number;
    label?: string;
    value?: number;
    status?: number;
  },
) =>
  requestClient.get<PageResult<DictionaryItem>>(
    `${prefix}getSysDictionaryInfoList`,
    { params },
  );
export const getDictionaryItem = (id: number) =>
  requestClient.get<DictionaryItem>(
    `${prefix}getSysDictionaryInfoListDetailsById`,
    { params: { id } },
  );
export const createDictionaryItem = (data: DictionaryItem) =>
  requestClient.post(`${prefix}createSysDictionaryInfo`, data);
export const updateDictionaryItem = (data: DictionaryItem) =>
  requestClient.put(`${prefix}updateSysDictionaryInfo`, data);
export const deleteDictionaryItem = (id: number) =>
  requestClient.delete(`${prefix}deleteSysDictionaryInfo`, { data: { id } });
