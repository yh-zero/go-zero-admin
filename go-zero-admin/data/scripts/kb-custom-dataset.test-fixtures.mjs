import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';

export const customRaw = readFileSync(new URL('../../application/ai/rpc/internal/logic/aiservice/testdata/custom-quality-v1.json', import.meta.url), 'utf8');
export const frontendCustomRaw = readFileSync(new URL('../../application/ai/rpc/internal/logic/aiservice/testdata/custom-feedback-export-v1.json', import.meta.url), 'utf8');
const digest = value => createHash('sha256').update(value).digest('hex');
const changed = mutate => { const value = JSON.parse(customRaw); mutate(value); return JSON.stringify(value); };

// 同一份坏样本经过浏览器导入与CLI入口，防止两套预检再次漂移。
export function invalidCustomSamples() {
  const number = (field, lexical) => customRaw.replace(new RegExp(`("${field}"\\s*:\\s*)\\d+`), `$1${lexical}`);
  return [
    ['根额外字段', changed(v => { v.provenance = {}; })],
    ['题目额外字段', changed(v => { v.cases[0].provenance = {}; })],
    ['空白说明', changed(v => { v.description = ' \t'; })],
    ['空注释', changed(v => { v.annotation = null; })],
    ['空元数据', changed(v => { v.metadata = null; })],
    ['尾下划线字段', changed(v => { v.metadata = { bad_: true }; })],
    ['连续下划线字段', changed(v => { v.metadata = { bad__field: true }; })],
    ['构造字段', changed(v => { v.metadata = { constructor: true }; })],
    ['原型字段', changed(v => { v.metadata = { prototype: true }; })],
    ['大小写字段', changed(v => { v.cases[0].Query = '被覆盖'; })],
    ['重复解码字段', customRaw.replace('"query":', '"qu\\u0065ry":"被覆盖","query":')],
    ['小数版本', number('version', '1.0')],
    ['指数版本', number('version', '1e0')],
    ['小数文档ID', number('source_doc_id', '10.0')],
    ['指数文档版本', number('source_doc_version', '2e0')],
    ['小数检查索引', number('required_group', '0.0')],
    ['指数检查索引', number('required_group', '0e0')],
    ['大文档ID', number('source_doc_id', '9007199254740992')],
    ['重复文档ID', changed(v => { v.documents.push({ ...v.documents[0], key: 'other', source: 'inline:other' }); })],
    ['长文档key', changed(v => { v.documents[0].key = 'x'.repeat(129); v.documents[0].source = `inline:${v.documents[0].key}`; })],
    ['长文档组', changed(v => { v.documents[0].kb = 'x'.repeat(65); })],
    ['长标题字节', changed(v => { v.documents[0].title = '中'.repeat(171); })],
    ['长题目ID', changed(v => { v.cases[0].id = 'x'.repeat(129); })],
    ['未声明负例组', changed(v => { Object.assign(v.cases[0], { kb: 'missing', expected_documents: [], evidence: [], checks: [] }); Object.assign(v.cases[0].answer, { kind: 'unanswerable', required_groups: [] }); })],
    ['伪造必需事实', changed(v => { v.cases[0].answer.required_groups[0] = ['原文不存在']; })],
    ['伪造证据', changed(v => { v.cases[0].evidence[0].groups[0] = ['原文不存在']; })],
    ['空白预期答案', changed(v => { v.cases[0].expected_answer_example = ' '; })],
    ['空白来源题目', changed(v => { v.cases[0].source_case_id = ' '; })],
    ['正文上限', changed(v => { const d = v.documents[0]; d.content = 'x'.repeat(65537); d.sha256 = digest(d.content); })],
    ['文档数量', changed(v => { v.documents = Array.from({ length: 101 }, (_, i) => ({ ...v.documents[0], key: `d${i}`, source: `inline:d${i}`, source_doc_id: i + 1 })); })],
    ['孤立代理项', changed(v => { v.cases[0].query = '\ud800'; })],
    ['深度上限', changed(v => { let nested = true; for (let i = 0; i < 64; i++) nested = [nested]; v.metadata = { nested }; })],
    ['快照上限', customRaw + ' '.repeat(262145)],
  ];
}

export function validCustomVariants() {
  return [customRaw, frontendCustomRaw, changed(v => { v.annotation = {}; delete v.description; }),
    changed(v => { v.cases[0].expected_answer_example = '字'.repeat(1500); }),
    changed(v => { const d = v.documents[0]; d.content = '预算五万元。保存３０天。'; d.sha256 = digest(d.content); }),
    customRaw.replace(/("required_group"\s*:\s*)0/u, '$1-0'),
    customRaw.replace('"metadata": {', '"metadata": {"large_number":1e10000,')];
}
