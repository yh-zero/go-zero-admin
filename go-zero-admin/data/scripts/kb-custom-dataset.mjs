// 浏览器与CLI只处理本地快照；字段、数字词法与深度对应Go自定义v1校验。
const maxBytes = 256 * 1024;
const encoder = new TextEncoder();
const object = value => value !== null && typeof value === 'object' && !Array.isArray(value);
const text = (value, limit = maxBytes) => typeof value === 'string' && !!value.trim() && encoder.encode(value).length <= limit;
const digest = value => typeof value === 'string' && /^[a-f0-9]{64}$/u.test(value);
const normalize = value => value.normalize('NFKC').replace(/\s/gu, '');
const strings = value => Array.isArray(value) && value.every(item => text(item));
const groups = value => Array.isArray(value) && value.every(group => strings(group) && group.length > 0);
const fail = message => { throw new Error(message); };
const wellFormed = value => !/[\uD800-\uDBFF](?![\uDC00-\uDFFF])|(?<![\uD800-\uDBFF])[\uDC00-\uDFFF]/u.test(value);

export function decodeCustomDataset(bytes) {
  if (!(bytes instanceof ArrayBuffer) && !ArrayBuffer.isView(bytes)) fail('题集必须是UTF-8字节');
  if (!bytes.byteLength || bytes.byteLength > maxBytes) fail('冻结题集最多256 KiB');
  let raw;
  try { raw = new TextDecoder('utf-8', { fatal: true, ignoreBOM: true }).decode(bytes); }
  catch { fail('题集不是有效UTF-8'); }
  // BOM不属于JSON语法，不能先剥掉再计算与上传文件不同的SHA。
  if (raw.startsWith('\ufeff')) fail('题集JSON不能包含UTF-8 BOM');
  return raw;
}

function parseStrictJSON(raw) {
  if (typeof raw !== 'string' || !raw.length || encoder.encode(raw).length > maxBytes || !wellFormed(raw)) fail('题集大小或Unicode无效');
  let position = 0;
  const numbers = new WeakMap();
  const whitespace = () => { while (/[\x20\t\r\n]/u.test(raw[position] ?? '') && position < raw.length) position++; };
  const expect = character => { whitespace(); if (raw[position++] !== character) fail('题集不是完整有效JSON'); };
  const readString = () => {
    whitespace(); if (raw[position] !== '"') fail('题集JSON字符串无效');
    const start = position++;
    while (position < raw.length) {
      const character = raw[position++];
      if (character === '\\') position++;
      else if (character === '"') {
        let value; try { value = JSON.parse(raw.slice(start, position)); } catch { fail('题集JSON字符串无效'); }
        if (!wellFormed(value)) fail('题集包含孤立Unicode代理项');
        return value;
      }
    }
    fail('题集JSON字符串不完整');
  };
  const read = depth => {
    if (depth > 64) fail('题集嵌套超过64层');
    whitespace(); const character = raw[position];
    if (character === '"') return { value: readString() };
    if (character === '{' || character === '[') {
      const isObject = character === '{', value = isObject ? {} : [], tokens = new Map();
      position++; whitespace();
      const end = isObject ? '}' : ']';
      if (raw[position] !== end) while (true) {
        const key = isObject ? readString() : value.length;
        if (isObject) {
          if (!/^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$/u.test(key) || key === 'constructor' || key === 'prototype' || Object.hasOwn(value, key)) fail('题集字段重复或不符合小写约定');
          expect(':');
        }
        const child = read(depth + 1);
        value[key] = child.value;
        if (child.number !== undefined) tokens.set(key, child.number);
        whitespace(); if (raw[position] !== ',') break;
        position++;
      }
      expect(end); numbers.set(value, tokens); return { value };
    }
    for (const [token, value] of [['true', true], ['false', false], ['null', null]]) if (raw.startsWith(token, position)) { position += token.length; return { value }; }
    const number = raw.slice(position).match(/^-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?/u)?.[0];
    if (!number) fail('题集JSON值无效');
    position += number.length; return { value: Number(number), number };
  };
  const { value } = read(0); whitespace();
  if (position !== raw.length) fail('题集不是完整有效JSON');
  return { value, numbers };
}

function fields(value, allowed, required = []) {
  return object(value) && Object.keys(value).every(key => allowed.includes(key)) && required.every(key => Object.hasOwn(value, key) && value[key] !== null);
}

async function sha256(value) {
  const sum = await globalThis.crypto.subtle.digest('SHA-256', encoder.encode(value));
  return [...new Uint8Array(sum)].map(byte => byte.toString(16).padStart(2, '0')).join('');
}

function inTexts(required, texts) {
  return required.every(group => group.some(alternative => texts.some(value => value.includes(normalize(alternative)))));
}

export async function parseCustomDataset(raw) {
  const { value: dataset, numbers } = parseStrictJSON(raw);
  if (!fields(dataset, ['version', 'description', 'annotation', 'metadata', 'documents', 'cases'], ['version', 'documents', 'cases']) || numbers.get(dataset)?.get('version') !== '1'
    || !Array.isArray(dataset.documents) || dataset.documents.length < 1 || dataset.documents.length > 100 || !Array.isArray(dataset.cases) || dataset.cases.length < 1 || dataset.cases.length > 100) fail('自定义题集结构或数量无效');
  for (const field of ['annotation', 'metadata']) if (Object.hasOwn(dataset, field) && !object(dataset[field])) fail('题集注释与元数据必须是对象');
  if (Object.hasOwn(dataset, 'description') && !text(dataset.description)) fail('题集说明不能为空');
  const integer = (row, key, minimum) => /^-?(?:0|[1-9]\d*)$/u.test(numbers.get(row)?.get(key) ?? '') && Number.isSafeInteger(row[key]) && row[key] >= minimum;
  const documents = new Map(), documentIDs = new Set(), kbGroups = new Set();
  for (const doc of dataset.documents) {
    if (!fields(doc, ['key', 'kb', 'source', 'title', 'sha256', 'source_doc_id', 'source_doc_version', 'content'], ['key', 'kb', 'source', 'title', 'sha256', 'source_doc_id', 'source_doc_version', 'content'])
      || !text(doc.key, 128) || documents.has(doc.key) || !text(doc.kb, 64) || !text(doc.title, 512) || doc.source !== `inline:${doc.key}` || !digest(doc.sha256)
      || !integer(doc, 'source_doc_id', 1) || !integer(doc, 'source_doc_version', 1) || documentIDs.has(doc.source_doc_id) || !text(doc.content, 65536)) fail('自定义内联资料的ID、版本、正文或字段无效');
    if (await sha256(doc.content) !== doc.sha256) fail('内联资料正文SHA256不符');
    documents.set(doc.key, doc); documentIDs.add(doc.source_doc_id); kbGroups.add(doc.kb);
  }
  const caseIDs = new Set();
  for (const row of dataset.cases) {
    if (!fields(row, ['id', 'kb', 'query', 'split', 'category', 'rationale', 'expected_documents', 'evidence', 'answer', 'checks', 'expected_answer_example', 'source_case_id'], ['id', 'kb', 'query', 'split', 'category', 'rationale', 'expected_documents', 'evidence', 'answer'])
      || !text(row.id, 128) || caseIDs.has(row.id) || !text(row.kb, 64) || !kbGroups.has(row.kb) || !text(row.query, 2000) || !text(row.category) || !text(row.rationale)
      || !['development', 'validation'].includes(row.split)) fail('题目字段、分组或标识无效');
    caseIDs.add(row.id);
    for (const field of ['expected_answer_example', 'source_case_id']) if (Object.hasOwn(row, field) && !text(row[field])) fail('题目预期答案或来源标识不能为空');
    if (!strings(row.expected_documents) || new Set(row.expected_documents).size !== row.expected_documents.length || row.expected_documents.some(key => documents.get(key)?.kb !== row.kb)) fail('预期资料缺失、重复或分组不符');
    const answer = row.answer;
    if (!fields(answer, ['kind', 'required_groups', 'forbidden_phrases', 'absence_policy'], ['kind', 'required_groups', 'forbidden_phrases']) || !['answerable', 'unanswerable'].includes(answer.kind)
      || !groups(answer.required_groups) || !strings(answer.forbidden_phrases) || Object.hasOwn(answer, 'absence_policy') && !['required', 'forbidden'].includes(answer.absence_policy) || !Array.isArray(row.evidence)) fail('题目答案规则无效');
    const expected = new Set(row.expected_documents);
    if (answer.kind === 'unanswerable') {
      if (expected.size || row.evidence.length || answer.required_groups.length) fail('负例不能填预期资料、证据或必需事实');
    } else if (!expected.size || !answer.required_groups.length || row.evidence.length !== expected.size || !inTexts(answer.required_groups, [...expected].map(key => normalize(documents.get(key).content)))) fail('必需事实没有对应内联依据');
    const evidenceDocs = new Set();
    for (const evidence of row.evidence) {
      if (!fields(evidence, ['document', 'groups'], ['document', 'groups']) || !text(evidence.document, 128) || !expected.has(evidence.document) || evidenceDocs.has(evidence.document)
        || !groups(evidence.groups) || !evidence.groups.length || !inTexts(evidence.groups, [normalize(documents.get(evidence.document).content)])) fail('证据组没有对应内联原文');
      evidenceDocs.add(evidence.document);
    }
    if (Object.hasOwn(row, 'checks')) {
      if (!Array.isArray(row.checks) || row.checks.length !== answer.required_groups.length) fail('检查项数量无效');
      const checked = new Set();
      for (const check of row.checks) {
        if (!fields(check, ['label', 'kind', 'required_group'], ['label', 'kind', 'required_group']) || !text(check.label) || !['fact', 'condition', 'risk', 'unknown'].includes(check.kind)
          || !integer(check, 'required_group', 0) || check.required_group >= answer.required_groups.length || checked.has(check.required_group)) fail('检查项索引或字段无效');
        checked.add(check.required_group);
      }
    }
  }
  return dataset;
}
