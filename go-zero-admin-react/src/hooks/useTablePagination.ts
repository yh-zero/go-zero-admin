import { useMemo } from 'react';
import type { Dispatch, SetStateAction } from 'react';
import type { PageParams } from '../api/types';

// 统一表格分页配置：页码变更走函数式更新，避免闭包捕获旧筛选条件导致分页错位。
export function useTablePagination<Params extends PageParams>(
  params: Params,
  setParams: Dispatch<SetStateAction<Params>>,
  total: number,
) {
  return useMemo(() => ({
    current: params.page,
    pageSize: params.page_size,
    total,
    showSizeChanger: true,
    // antd 回调参数名为 pageSize，契约字段为 page_size，此处做一次映射。
    onChange: (page: number, pageSize: number) => setParams(current => ({ ...current, page, page_size: pageSize })),
  }), [params, setParams, total]);
}
