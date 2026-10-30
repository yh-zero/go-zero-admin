import { useEffect } from 'react';
import type { Dispatch, SetStateAction } from 'react';
import { useQuery } from '@tanstack/react-query';
import type { QueryKey } from '@tanstack/react-query';
import type { PageParams, PageResult } from '../api/types';

export function correctPageParams<Params extends PageParams>(params: Params, total: number): Params {
  const lastPage = Math.max(1, Math.ceil(total / (params.page_size ?? 20)));
  return (params.page ?? 1) > lastPage ? { ...params, page: lastPage } : params;
}

interface PagedQueryOptions<Params extends PageParams, Item> {
  params: Params;
  setParams: Dispatch<SetStateAction<Params>>;
  queryKey: QueryKey;
  queryFn: () => Promise<PageResult<Item>>;
  enabled?: boolean;
}

export function usePagedQuery<Params extends PageParams, Item>({ params, setParams, enabled,
  ...options }: PagedQueryOptions<Params, Item>) {
  const query = useQuery({ ...options, enabled });
  const total = query.isSuccess && !query.isFetching ? query.data.total : undefined;
  useEffect(() => {
    if (enabled === false || total === undefined) return;
    // 旧查询完成时，不能用它的总数修改用户刚切换的筛选或每页条数。
    setParams((current) => current === params ? correctPageParams(current, total) : current);
  }, [enabled, params, setParams, total]);
  return query;
}
