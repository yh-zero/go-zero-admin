import { createHash } from 'node:crypto';
import { lstat, mkdir, readFile, realpath, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { loadDataset } from './eval-kb-quality.mjs';

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../..');
const sources = [
  { name: 'business-acceptance-cases.json', sha256: '176e20b4563d8c892b375fe45af89a4a27b78ee1d62c9686e9b49b3956696760' },
  { name: 'business-cases.json', sha256: 'f3a4016eabc10a974e3acffc6d7347477976d00923c196c608de55a6db99538b' },
  { name: 'business-failure-regression-cases.json', sha256: '6f6fbf2a4881751d1a9a77f2cde4571c2176cd3cca26fdab310f3a6eed0c9b87' },
  { name: 'business-evidence-cases.json', sha256: '1b51cb93a575876002bd9877c4ffa2ef3ab95f43785bd5ea4a20292c8b6eb5ca' },
];
const hash = bytes => createHash('sha256').update(bytes).digest('hex');

// 检查项只给既有事实组加中文标签，不引入第二套评分或伪造未知实例值。
export function validateDatasetChecks(dataset) {
  const required = dataset.annotation?.checks_version === 1;
  for (const row of dataset.cases) {
    if (row.checks === undefined && !required) continue;
    if (!Array.isArray(row.checks) || row.checks.length !== row.answer.required_groups.length) throw new Error('检查项数量与事实组不一致');
    const seen = new Set();
    for (const item of row.checks) {
      if (!item || typeof item.label !== 'string' || !item.label.trim() || item.label !== item.label.trim() || Buffer.byteLength(item.label) > 256
        || !['fact', 'condition', 'risk', 'unknown'].includes(item.kind) || !Number.isSafeInteger(item.required_group)
        || item.required_group < 0 || item.required_group >= row.answer.required_groups.length || seen.has(item.required_group)) throw new Error('检查项字段或事实组映射无效');
      seen.add(item.required_group);
    }
  }
}

async function readSources(root) {
  const result = [];
  for (const source of sources) {
    const path = resolve(root, 'ai_document/ai-rag', source.name);
    if ((await lstat(path)).isSymbolicLink() || await realpath(path) !== path) throw new Error('题集来源不允许目录链接或符号链接');
    const bytes = await readFile(path);
    if (source.sha256 && hash(bytes) !== source.sha256) throw new Error(`冻结题集已变更：${source.name}`);
    try { validateDatasetChecks(await loadDataset(path)); }
    catch { throw new Error(`题集原文、证据或检查项校验失败：${source.name}`); }
    if (!bytes.equals(await readFile(path))) throw new Error(`同步中题集发生变化：${source.name}`);
    result.push({ ...source, bytes, sha256: hash(bytes) });
  }
  return result;
}

// 目标固定为四份go:embed数据；--check不创建目录、不改写文件。
export async function syncEvaluationData({ check = false } = {}) {
  if (typeof check !== 'boolean') throw new Error('check参数必须为布尔值');
  const root = await realpath(repositoryRoot);
  const inputs = await readSources(root);
  const parent = resolve(root, 'go-zero-admin/application/ai/rpc/internal/logic/aiservice');
  if (await realpath(parent) !== parent) throw new Error('内嵌数据父目录不允许目录链接');
  const destination = resolve(parent, 'evaluation_data');
  let directory;
  try { directory = await lstat(destination); }
  catch (error) {
    if (error.code !== 'ENOENT') throw error;
    if (check) throw new Error('内嵌数据目录不存在，请先执行同步');
    await mkdir(destination);
    directory = await lstat(destination);
  }
  if (!directory.isDirectory() || directory.isSymbolicLink() || await realpath(destination) !== destination) throw new Error('内嵌数据目标必须为固定真实目录');
  const targets = [];
  for (const source of inputs) {
    const path = resolve(destination, source.name);
    let current;
    try {
      const info = await lstat(path);
      if (!info.isFile() || info.isSymbolicLink() || info.nlink !== 1) throw new Error('内嵌数据文件不允许目录链接、符号链接或硬链接');
      current = await readFile(path);
    } catch (error) { if (error.code !== 'ENOENT') throw error; }
    const matches = current?.equals(source.bytes) === true;
    if (check && !matches) throw new Error(`内嵌数据与源JSON不一致：${source.name}`);
    targets.push({ ...source, path, matches });
  }
  const files = [];
  for (const target of targets) {
    const written = !target.matches && !check;
    if (written) {
      await writeFile(target.path, target.bytes);
      if (!(await readFile(target.path)).equals(target.bytes)) throw new Error(`同步后字节校验失败：${target.name}`);
    }
    files.push({ name: target.name, sha256: target.sha256, bytes: target.bytes.length, written });
  }
  return { mode: check ? 'check' : 'sync', http_requests: 0, files };
}

async function main(args) {
  if (args.length > 1 || (args.length === 1 && args[0] !== '--check')) throw new Error('只支持无参数同步或--check只读校验');
  console.log(JSON.stringify(await syncEvaluationData({ check: args[0] === '--check' })));
}

if (process.argv[1] && pathToFileURL(resolve(process.argv[1])).href === import.meta.url) {
  main(process.argv.slice(2)).catch(error => { console.error(error.message); process.exitCode = 1; });
}
