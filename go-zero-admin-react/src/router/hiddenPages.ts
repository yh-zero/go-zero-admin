export const hiddenPageLinks = [
  { name: 'ai_workflow_editor', path: '/ai/workflows/new', title: '新建常用流程', menuPath: '/ai/workflows' },
  { name: 'ai_workflow_editor', path: '/ai/workflows/:id/edit', title: '编辑常用流程', menuPath: '/ai/workflows' },
  { name: 'ai_knowledge_workspace', path: '/ai/knowledge/:id', title: '知识库工作区', menuPath: '/ai/knowledge' },
  { name: 'ai_knowledge_quality', path: '/ai/knowledge/:id/quality', title: '知识库质量验收', menuPath: '/ai/knowledge' },
] as const;

export function hiddenPageForPath(path: string) {
  return hiddenPageLinks.find(page => page.path === path || page.path === '/ai/workflows/:id/edit' && /^\/ai\/workflows\/[1-9]\d*\/edit$/.test(path)
    || page.path === '/ai/knowledge/:id' && /^\/ai\/knowledge\/[1-9]\d*$/.test(path)
    || page.path === '/ai/knowledge/:id/quality' && /^\/ai\/knowledge\/[1-9]\d*\/quality$/.test(path));
}
