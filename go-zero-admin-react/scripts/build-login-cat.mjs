import { readFile, writeFile } from 'node:fs/promises';
import { basename, dirname, extname, join, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { decodeRgbaPng, splitPngAtlas } from './pngSpriteAtlas.mjs';

// 格式和字段编号来自 Rive 官方公开运行库；此生成器不复制第三方作品模板。
// https://github.com/rive-app/rive-runtime/blob/main/include/rive/runtime_header.hpp
// https://github.com/rive-app/rive-runtime/tree/main/include/rive/generated
const uint = value => {
  if (!Number.isSafeInteger(value) || value < 0) throw new Error(`Invalid uint: ${value}`);
  const result = [];
  do { const byte = value % 128; value = Math.floor(value / 128); result.push(byte | (value ? 128 : 0)); } while (value);
  return Buffer.from(result);
};
const float = value => { const buffer = Buffer.alloc(4); buffer.writeFloatLE(value); return buffer; };
const bytes = value => Buffer.concat([uint(value.length), value]);
const text = value => bytes(Buffer.from(value, 'utf8'));
const property = (key, value) => Buffer.concat([uint(key), value]);
const object = (type, ...properties) => Buffer.concat([uint(type), ...properties, uint(0)]);
const u = (key, value) => property(key, uint(value));
const f = (key, value) => property(key, float(value));
const s = (key, value) => property(key, text(value));

function keyframes(objectId, propertyKey, frames) {
  return [object(25, u(51, objectId)), object(26, u(53, propertyKey)),
    ...frames.map(([frame, value]) => object(30, u(67, frame), u(68, 1), f(70, value)))];
}

const imageNames = ['idle', 'look-left', 'look-right', 'cover-start', 'cover', 'cover-end', 'success', 'failure', 'blink'];
const blinkTrack = [[0, 0], [132, 0], [134, 1], [138, 1], [140, 0], [300, 0], [302, 1], [306, 1], [308, 0], [360, 0]];
const fadeTo = (first, second) => [[0, first], [8, first], [11, second]];
const animationSpecs = [
  { name: 'idle', duration: 360, loop: true, image: 0, tracks: { 0: blinkTrack.map(([frame, value]) => [frame, 1 - value]), 8: blinkTrack } },
  { name: 'look_left', duration: 360, loop: true, image: 1 },
  { name: 'look_right', duration: 360, loop: true, image: 2 },
  { name: 'hands_up', duration: 30, image: 4, tracks: { 3: fadeTo(1, 0), 4: fadeTo(0, 1) } },
  { name: 'hands_down', duration: 24, image: 0, tracks: { 5: fadeTo(1, 0), 0: fadeTo(0, 1) } },
  { name: 'success', duration: 60, image: 6 },
  { name: 'fail', duration: 60, image: 7 },
];

function animation(spec, width, height) {
  const { name, duration, loop, image, tracks = {} } = spec;
  const result = [object(31, s(55, name), u(56, 60), u(57, duration), u(59, loop ? 1 : 0))];
  for (let index = 0; index < imageNames.length; index++) {
    result.push(...keyframes(index + 2, 18, tracks[index] ?? [[0, Number(index === image)]]));
  }
  const xTrack = name === 'fail' ? [[0, 0], [8, -4], [16, 4], [24, -2], [36, 0], [60, 0]] : [[0, 0]];
  const yTrack = loop ? [[0, 0], [90, -2], [180, 0], [270, -2], [360, 0]] : [[0, 0], [20, name === 'success' ? -5 : 0], [duration, 0]];
  result.push(...keyframes(1, 13, xTrack.map(([frame, value]) => [frame, width / 2 + value])));
  result.push(...keyframes(1, 14, yTrack.map(([frame, value]) => [frame, height / 2 + value])));
  for (const axis of [16, 17]) result.push(...keyframes(1, axis, name === 'success' ? [[0, 1], [20, 1.025], [60, 1]] : [[0, 1]]));
  return result;
}

function transition(to, conditions = [], exit = false) {
  // 短混合仅平滑相邻姿态，避免较长交叉淡化产生双影；出口等待当前动画完整播放。
  const output = [object(65, u(151, to), u(158, 40), u(152, 32 | (exit ? 12 : 0)), ...(exit ? [u(160, 100)] : []))];
  for (const condition of conditions) {
    if (condition.type === 'trigger') output.push(object(68, u(155, condition.input)));
    if (condition.type === 'bool') output.push(object(71, u(155, condition.input), u(156, condition.value ? 0 : 1)));
    if (condition.type === 'number') output.push(object(70, u(155, condition.input), u(156, condition.op), f(157, condition.value)));
  }
  return output;
}

function gazeTransitions(currentState) {
  const number = (op, value) => ({ type: 'number', input: 3, op, value });
  const directions = [[3, [number(3, -0.25), number(2, 0.25)]], [4, [number(4, -0.25)]], [5, [number(5, 0.25)]]];
  return [
    ...transition(6, [{ type: 'bool', input: 2, value: true }]),
    ...directions.filter(([state]) => state !== currentState).flatMap(([state, conditions]) => transition(state, conditions)),
  ];
}

function stateMachine() {
  return [
    object(53, s(55, 'Login Machine')),
    object(58, s(138, 'success')), object(58, s(138, 'fail')), object(59, s(138, 'is_hands_up')), object(56, s(138, 'look_direction')),
    object(57, s(138, 'Login')),
    object(63), ...transition(3),
    object(62), ...transition(8, [{ type: 'trigger', input: 0 }]), ...transition(9, [{ type: 'trigger', input: 1 }]),
    object(64),
    object(61, u(149, 0)), ...gazeTransitions(3),
    object(61, u(149, 1)), ...gazeTransitions(4),
    object(61, u(149, 2)), ...gazeTransitions(5),
    object(61, u(149, 3)), ...transition(7, [{ type: 'bool', input: 2, value: false }]),
    object(61, u(149, 4)), ...transition(6, [{ type: 'bool', input: 2, value: true }]), ...transition(3, [], true),
    object(61, u(149, 5)), ...transition(3, [], true),
    object(61, u(149, 6)), ...transition(3, [], true),
  ];
}

export async function buildLoginCat(imagePaths, outputPath) {
  if (imagePaths.length !== 1 && imagePaths.length !== imageNames.length) throw new Error(`需要一张 3×3 PNG 图集，或依次提供九张 PNG：${imageNames.join('、')}`);
  if (extname(outputPath).toLowerCase() !== '.riv') throw new Error('动画输出路径必须以 .riv 结尾');
  const fallbackPath = join(dirname(outputPath), `${basename(outputPath, extname(outputPath))}.png`);
  if (imagePaths.some(path => resolve(path) === resolve(outputPath) || resolve(path) === resolve(fallbackPath))) throw new Error('输出文件不能覆盖输入素材');
  const sources = await Promise.all(imagePaths.map(path => readFile(path)));
  const images = sources.length === 1 ? splitPngAtlas(sources[0]) : sources;
  const sizes = images.map(decodeRgbaPng);
  const { width, height } = sizes[0];
  if (sizes.some(size => size.width !== width || size.height !== height)) throw new Error('九张 PNG 必须尺寸一致并对齐');
  const chunks = [Buffer.from('RIVE'), uint(7), uint(0), uint(0), uint(0), object(23)];
  images.forEach((image, index) => {
    chunks.push(object(105, s(203, `${imageNames[index]}.png`), u(204, index + 1), f(208, width), f(207, height)));
    chunks.push(object(106, property(212, bytes(image))));
  });
  chunks.push(object(1, s(4, 'cat_artboard'), f(7, width), f(8, height), u(236, 0)));
  chunks.push(object(2, s(4, 'cat-root'), u(5, 0), f(13, width / 2), f(14, height / 2)));
  images.forEach((_, index) => chunks.push(object(100, s(4, imageNames[index]), u(5, 1), u(206, index), f(18, Number(index === 0)))));
  animationSpecs.forEach(spec => chunks.push(...animation(spec, width, height)));
  chunks.push(...stateMachine());
  const buffer = Buffer.concat(chunks);
  await writeFile(outputPath, buffer);
  await writeFile(fallbackPath, images[0]);
  return { bytes: buffer.length, width, height, fallbackPath };
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  const [outputPath, ...imagePaths] = process.argv.slice(2);
  if (!outputPath) throw new Error(`用法：node scripts/build-login-cat.mjs output.riv atlas.png；或依次传入 ${imageNames.map(name => `${name}.png`).join(' ')}`);
  console.log(await buildLoginCat(imagePaths, outputPath));
}
