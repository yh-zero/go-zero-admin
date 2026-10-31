import { deflateSync, inflateSync } from 'node:zlib';

// 仅支持项目素材需要的 RGBA8 非交错 PNG；机械裁切保留颜色通道与透明度。
// 格式依据：https://www.w3.org/TR/png-3/
const signature = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
const colorMetadata = new Set(['cHRM', 'gAMA', 'iCCP', 'sBIT', 'sRGB', 'cICP']);
const crcTable = Array.from({ length: 256 }, (_, index) => {
  let crc = index;
  for (let bit = 0; bit < 8; bit++) crc = (crc >>> 1) ^ ((crc & 1) ? 0xedb88320 : 0);
  return crc >>> 0;
});

function crc32(buffer) {
  let crc = 0xffffffff;
  for (const byte of buffer) crc = crcTable[(crc ^ byte) & 255] ^ (crc >>> 8);
  return (crc ^ 0xffffffff) >>> 0;
}

function chunk(type, data) {
  const body = Buffer.concat([Buffer.from(type, 'ascii'), data]);
  const header = Buffer.alloc(4);
  header.writeUInt32BE(data.length);
  const checksum = Buffer.alloc(4);
  checksum.writeUInt32BE(crc32(body));
  return Buffer.concat([header, body, checksum]);
}

function readChunks(buffer) {
  if (buffer.length < 33 || !buffer.subarray(0, 8).equals(signature)) throw new Error('素材必须为完整 PNG');
  const chunks = [];
  let offset = 8;
  while (offset < buffer.length) {
    if (offset + 12 > buffer.length) throw new Error('PNG chunk 被截断');
    const length = buffer.readUInt32BE(offset);
    const end = offset + 12 + length;
    if (end > buffer.length) throw new Error('PNG chunk 长度越界');
    const type = buffer.toString('latin1', offset + 4, offset + 8);
    if (!/^[A-Za-z]{4}$/.test(type) || type[2] !== type[2].toUpperCase()) throw new Error('PNG chunk 类型无效');
    if (crc32(buffer.subarray(offset + 4, end - 4)) !== buffer.readUInt32BE(end - 4)) throw new Error(`PNG ${type} CRC 校验失败`);
    chunks.push({ type, data: buffer.subarray(offset + 8, end - 4) });
    offset = end;
    if (type === 'IEND') {
      if (length || offset !== buffer.length) throw new Error('PNG IEND 或尾部数据无效');
      break;
    }
  }
  if (chunks[0]?.type !== 'IHDR' || chunks.at(-1)?.type !== 'IEND') throw new Error('PNG 缺少 IHDR/IEND');
  return chunks;
}

function imageSize(width, height) {
  if (!Number.isInteger(width) || !Number.isInteger(height) || width <= 0 || height <= 0 || width * height > 16_777_216) {
    throw new Error('PNG 尺寸无效或超过 16777216 像素上限');
  }
  return width * height * 4;
}

function paeth(left, up, corner) {
  const prediction = left + up - corner;
  const a = Math.abs(prediction - left), b = Math.abs(prediction - up), c = Math.abs(prediction - corner);
  return a <= b && a <= c ? left : b <= c ? up : corner;
}

export function decodeRgbaPng(buffer) {
  const chunks = readChunks(buffer);
  const header = chunks[0].data;
  if (header.length !== 13 || chunks.filter(item => item.type === 'IHDR').length !== 1) throw new Error('PNG IHDR 无效');
  const width = header.readUInt32BE(0), height = header.readUInt32BE(4);
  const size = imageSize(width, height);
  if (header[8] !== 8 || header[9] !== 6 || header[10] !== 0 || header[11] !== 0 || header[12] !== 0) {
    throw new Error('仅支持 RGBA8、标准压缩和非交错 PNG');
  }
  const idat = [], metadata = [];
  let endedData = false;
  for (const entry of chunks.slice(1)) {
    if (entry.type === 'IDAT') {
      if (endedData) throw new Error('PNG IDAT 必须连续');
      idat.push(entry.data);
    } else {
      if (idat.length) endedData = true;
      if (entry.type === 'PLTE' && (idat.length || !entry.data.length || entry.data.length % 3 || entry.data.length > 768)) throw new Error('PNG PLTE 无效');
      if (entry.type[0] === entry.type[0].toUpperCase() && !['PLTE', 'IEND'].includes(entry.type)) throw new Error(`不支持 PNG 关键 chunk：${entry.type}`);
      if (['acTL', 'fcTL', 'fdAT'].includes(entry.type)) throw new Error('素材必须为静态 PNG');
      if (colorMetadata.has(entry.type)) {
        if (idat.length) throw new Error('PNG 色彩元数据必须位于 IDAT 之前');
        metadata.push(entry);
      }
    }
  }
  if (!idat.length) throw new Error('PNG 缺少像素数据');
  const stride = width * 4;
  const expected = size + height;
  const raw = inflateSync(Buffer.concat(idat), { maxOutputLength: expected });
  if (raw.length !== expected) throw new Error('PNG 解压长度与尺寸不一致');
  const pixels = Buffer.alloc(size);
  for (let y = 0; y < height; y++) {
    const filter = raw[y * (stride + 1)];
    if (filter > 4) throw new Error(`PNG 行滤波类型无效：${filter}`);
    for (let x = 0; x < stride; x++) {
      const offset = y * stride + x;
      const left = x >= 4 ? pixels[offset - 4] : 0;
      const up = y > 0 ? pixels[offset - stride] : 0;
      const corner = y > 0 && x >= 4 ? pixels[offset - stride - 4] : 0;
      const prediction = filter === 0 ? 0 : filter === 1 ? left : filter === 2 ? up : filter === 3 ? Math.floor((left + up) / 2) : paeth(left, up, corner);
      pixels[offset] = (raw[y * (stride + 1) + x + 1] + prediction) & 255;
    }
  }
  return { width, height, pixels, metadata };
}

export function encodeRgbaPng({ width, height, pixels, metadata = [] }) {
  if (pixels.length !== imageSize(width, height)) throw new Error('RGBA 像素长度与尺寸不一致');
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width, 0); header.writeUInt32BE(height, 4); header[8] = 8; header[9] = 6;
  const stride = width * 4, raw = Buffer.alloc((stride + 1) * height);
  for (let y = 0; y < height; y++) pixels.copy(raw, y * (stride + 1) + 1, y * stride, (y + 1) * stride);
  return Buffer.concat([signature, chunk('IHDR', header), ...metadata.filter(entry => colorMetadata.has(entry.type)).map(entry => chunk(entry.type, entry.data)), chunk('IDAT', deflateSync(raw, { level: 9 })), chunk('IEND', Buffer.alloc(0))]);
}

export function splitPngAtlas(buffer) {
  const source = decodeRgbaPng(buffer);
  if (source.width % 3 || source.height % 3) throw new Error('图集宽高必须能被 3 整除，无法严格等分时请先对齐素材');
  const width = source.width / 3, height = source.height / 3;
  return Array.from({ length: 9 }, (_, index) => {
    const pixels = Buffer.alloc(width * height * 4);
    for (let y = 0; y < height; y++) {
      const start = ((Math.floor(index / 3) * height + y) * source.width + (index % 3) * width) * 4;
      source.pixels.copy(pixels, y * width * 4, start, start + width * 4);
    }
    return encodeRgbaPng({ width, height, pixels, metadata: source.metadata });
  });
}
