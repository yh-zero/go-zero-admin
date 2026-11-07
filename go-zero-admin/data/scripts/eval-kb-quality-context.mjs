export function validateContext(context) {
  const object = value => value !== null && typeof value === 'object' && !Array.isArray(value);
  const label = value => typeof value === 'string' && value.trim().length > 0;
  const positive = value => Number.isSafeInteger(value) && value > 0;
  if (!object(context) || typeof context.binary_sha !== 'string' || !/^[a-f0-9]{64}$/iu.test(context.binary_sha)) throw new Error('context需要64位十六进制构建binary_sha摘要');
  if (!object(context.model) || !label(context.model.provider) || !label(context.model.name)) throw new Error('context模型需要非空provider和name');
  if (!object(context.embedding) || !label(context.embedding.provider) || !label(context.embedding.model) || !positive(context.embedding.dimension)) throw new Error('context Embedding需要provider、model和正安全整数dimension');
  if (!object(context.config) || ['max_context_tokens', 'analysis_answer_token_cap', 'recall_top_k', 'return_top_k'].some(key => !positive(context.config[key]))
    || context.config.analysis_answer_token_cap > context.config.max_context_tokens || context.config.return_top_k > context.config.recall_top_k) throw new Error('context预算与TopK需要正安全整数，回答预算及返回TopK不能超过对应上限');
  const budgets = new Set(['max_context_tokens', 'analysis_answer_token_cap', 'context_tokens', 'max_tokens', 'answer_tokens']);
  function inspect(value) {
    if (!value || typeof value !== 'object') return;
    for (const [key, child] of Object.entries(value)) {
      if ((key === 'temperature' || key === 'analysis_temperature') && child !== 'provider_default'
        && !(typeof child === 'number' && Number.isFinite(child) && child >= 0)) throw new Error('context temperature需要非负有限数或provider_default');
      if ((!budgets.has(key) && /(key|token|password|secret|credential|authorization|cookie)/iu.test(key)) || /e-?mail/iu.test(key)) throw new Error('context包含敏感字段；只传入已消毒运行标签');
      if (typeof child === 'string' && (/Bearer\s|sk-[A-Za-z0-9]{12,}|eyJ[A-Za-z0-9_-]{12,}\./u.test(child) || /[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}/iu.test(child))) throw new Error('context包含疑似凭据或邮箱值');
      inspect(child);
    }
  }
  inspect(context); return context;
}
