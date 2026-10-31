import { useEffect } from 'react';
import { QueryClientProvider } from '@tanstack/react-query';
import { App as AntdApp, ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { RouterProvider } from 'react-router-dom';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import { router } from './router';
import { queryClient } from './utils/queryClient';
import { setRequestErrorReporter } from './utils/request';
import { appTheme } from './theme';

dayjs.locale('zh-cn');
function RequestFeedback() {
  const { message } = AntdApp.useApp();
  useEffect(() => {
    setRequestErrorReporter((content) => { void message.error({ content, key: 'request-error' }); });
    return () => setRequestErrorReporter();
  }, [message]);
  return null;
}
export default function App() {
  return <ConfigProvider locale={zhCN} theme={appTheme}><AntdApp><QueryClientProvider client={queryClient}>
    <RequestFeedback /><RouterProvider router={router} />
  </QueryClientProvider></AntdApp></ConfigProvider>;
}
