import { Alert, Tag } from 'antd';
import type { TaskAnalysis } from './workflowAnalysis';
import { CopyTextButton, SafeMarkdown } from './SafeMarkdown';
import styles from './workflowAnalysis.module.css';

export function WorkflowAnalysisResult({ analysis }: { analysis: TaskAnalysis }) {
  const { result } = analysis;
  return <section className={styles.result} aria-label={`第 ${analysis.step} 步分析结果`}>
    <div className={styles.heading}><strong>{analysis.name}</strong><Tag>第 {analysis.step} 步</Tag></div>
    {result.status === 'no_matches' && <Alert type="info" showIcon title="未匹配到资料，未调用聊天模型。" />}
    {result.status === 'truncated' && <Alert type="warning" showIcon title="分析达到输出上限，内容已截断。请缩小分析范围后重新运行。" />}
    {result.analysis && <><h3>分析内容 <CopyTextButton text={result.analysis} label="复制分析" /></h3><SafeMarkdown content={result.analysis} /></>}
    {!!result.sources.length && <><h3>参考资料</h3><p className={styles.note}>以下保留本次分析使用的资料片段，不代表完整原文；相似度不表示答案正确率。当前原文可能已更新，不代表执行当时的版本。</p>
      <ol className={styles.sources}>{result.sources.map(source => <li key={`${source.kb_id}:${source.doc_id}:${source.chunk_id}`}>
        <strong>{source.doc_title || `文档 #${source.doc_id}`}</strong>
        {' · '}<a href={`/ai/knowledge/${source.kb_id}?doc_id=${source.doc_id}`}>打开当前原文</a>
        <p className={styles.text}>{source.snippet}</p>
        <details className={styles.metadata}><summary>查看来源标识</summary><p>知识库 ID：{source.kb_id} · 文档 ID：{source.doc_id} · 分块 ID：{source.chunk_id} · 相似度：{source.score}</p></details>
      </li>)}</ol></>}
    <details className={styles.usage}><summary>已知用量{result.usage.complete ? '' : '（用量不完整）'}</summary>
      <p>聊天模型：输入 {result.usage.prompt_tokens} · 输出 {result.usage.completion_tokens} · 合计 {result.usage.total_tokens} tokens；不含 Embedding 用量。</p>
      {!result.usage.complete && <p>供应商未返回完整用量；这些数值不代表全部计费，0 不代表免费。</p>}
      {result.usage.complete && result.status !== 'no_matches' && <p>展示供应商返回的已知 token，不作为费用明细。</p>}
    </details>
  </section>;
}
