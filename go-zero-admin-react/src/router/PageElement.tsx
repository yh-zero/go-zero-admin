import { createElement, Suspense } from 'react';
import { Spin } from 'antd';
import { pages } from './registry';
export default function PageElement({ name }: { name: keyof typeof pages }) {
  return <Suspense fallback={<Spin className="route-spinner" />}>{createElement(pages[name])}</Suspense>;
}
