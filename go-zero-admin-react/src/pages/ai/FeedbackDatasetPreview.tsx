import { Alert, Collapse, Tag } from 'antd';
import type { QualityCatalogue } from './knowledgeQuality';
import styles from './knowledgeFeedback.module.css';

export function FeedbackDatasetPreview({ catalogue }: { catalogue: QualityCatalogue }) {
  return <section className={styles.detail} aria-label="冻结题集预览">
    <Alert type="info" title="采纳不等于质量签署" description="预期答案仅供人工核对，不发送给模型。检查项是词法筛查，仍需复核内容。生成、导入与下载不会开始分析。" />
    {catalogue.dataset.cases.map(item => <article key={item.id}><Tag>{item.split === 'development' ? '开发题' : '初始预留题'}</Tag><h4>{item.query}</h4>
      <p>复核结论：{item.answer.kind === 'answerable' ? '资料支持回答' : '资料不足，需明确说明'}</p><h4>预期答案</h4><p className={styles.text}>{item.expected_answer_example ?? '未提供示例答案'}</p>
      <h4>检查项</h4>{item.answer.required_groups.length ? <ul>{item.answer.required_groups.map((group, index) => <li key={`${item.id}:${index}:${group.join('|')}`}>{group.join(' / ')}</li>)}</ul> : <p>核对资料不足说明及是否伪造答案。</p>}<p>{item.rationale}</p>
    </article>)}
    <Collapse items={catalogue.dataset.documents.map(doc => ({ key: doc.key, label: `${doc.title} · 原资料 #${doc.source_doc_id ?? '未记录'} · 版本 ${doc.source_doc_version ?? '未记录'}`, children: <><p className={styles.text}>正文指纹：{doc.sha256}</p><pre className={styles.code}>{doc.content ?? '此题集不含内联原文'}</pre></> }))} />
    <p className={styles.note}>{catalogue.dataset.description} · {new TextEncoder().encode(catalogue.raw).length} UTF-8 字节</p>
  </section>;
}
