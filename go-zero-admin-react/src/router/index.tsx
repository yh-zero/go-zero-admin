import { createBrowserRouter, Navigate } from 'react-router-dom';
import BasicLayout from '../layouts/BasicLayout';
import LoginPage from '../pages/login';
import { pageLinks } from './registry';
import RequireAuth from './RequireAuth';
import DynamicPage from './DynamicPage';
import PageElement from './PageElement';

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/', element: <RequireAuth><BasicLayout /></RequireAuth>, children: [
    { index: true, element: <Navigate to="/dashboard" replace /> },
    ...pageLinks.map(page => ({ path: page.path.slice(1), element: <PageElement name={page.name} /> })),
    { path: '*', element: <DynamicPage /> },
  ] },
]);
