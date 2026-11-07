export const hiddenPageLinks = [
  { name: 'ai_workflow_editor', path: '/ai/workflows/new', title: '新建工作流', menuPath: '/ai/workflows' },
  { name: 'ai_workflow_editor', path: '/ai/workflows/:id/edit', title: '编辑工作流', menuPath: '/ai/workflows' },
] as const;

export function hiddenPageForPath(path: string) {
  return hiddenPageLinks.find(page => page.path === path || page.path === '/ai/workflows/:id/edit' && /^\/ai\/workflows\/[1-9]\d*\/edit$/.test(path));
}
