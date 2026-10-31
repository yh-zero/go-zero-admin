import test from 'node:test';
import assert from 'node:assert/strict';
import { deflateSync } from 'node:zlib';
import { decodeRgbaPng, encodeRgbaPng, splitPngAtlas } from './pngSpriteAtlas.mjs';

// 夹具独立实现 PNG 包装与行预测，确保测试不仅是编码器/解码器彼此往返。
function fixtureChunk(type, data) {
  const buffer = Buffer.alloc(data.length + 12);
  buffer.writeUInt32BE(data.length);
  buffer.write(type, 4);
  data.copy(buffer, 8);
  let crc = 0xffffffff;
  for (const value of buffer.subarray(4, -4)) {
    crc ^= value;
    for (let i = 0; i < 8; i++) crc = (crc & 1) ? (crc >>> 1) ^ 0xedb88320 : crc >>> 1;
  }
  buffer.writeUInt32BE((crc ^ 0xffffffff) >>> 0, buffer.length - 4);
  return buffer;
}

function fixture({ width = 6, height = 15, badFilter, interlace = 0, bitDepth = 8, colorType = 6 } = {}) {
  const pixels = Buffer.from(Array.from({ length: width * height * 4 }, (_, index) => (index * 29 + Math.floor(index / (width * 4)) * 53) & 255));
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width); header.writeUInt32BE(height, 4);
  header[8] = bitDepth; header[9] = colorType; header[12] = interlace;
  const stride = width * 4, raw = Buffer.alloc((stride + 1) * height);
  for (let y = 0; y < height; y++) {
    const filter = y % 5;
    raw[y * (stride + 1)] = badFilter ?? filter;
    for (let x = 0; x < stride; x++) {
      const index = y * stride + x;
      const left = x < 4 ? 0 : pixels[index - 4];
      const up = y === 0 ? 0 : pixels[index - stride];
      const corner = y === 0 || x < 4 ? 0 : pixels[index - stride - 4];
      let predict = 0;
      if (filter === 1) predict = left;
      if (filter === 2) predict = up;
      if (filter === 3) predict = Math.floor((left + up) / 2);
      if (filter === 4) {
        const value = left + up - corner;
        const candidates = [left, up, corner];
        predict = candidates.reduce((best, candidate) => Math.abs(value - candidate) < Math.abs(value - best) ? candidate : best);
      }
      raw[y * (stride + 1) + x + 1] = (pixels[index] - predict) & 255;
    }
  }
  const png = Buffer.concat([Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]), fixtureChunk('IHDR', header), fixtureChunk('gAMA', Buffer.from([0, 0, 177, 143])), fixtureChunk('IDAT', deflateSync(raw)), fixtureChunk('IEND', Buffer.alloc(0))]);
  return { png, pixels, width, height };
}

test('解码全部五种滤波并保留透明通道与隐藏 RGB', () => {
  const source = fixture();
  assert.deepEqual(decodeRgbaPng(source.png).pixels, source.pixels);
});

test('九格输出每个 RGBA 像素与原图对应位置一致', () => {
  const source = fixture();
  const crops = splitPngAtlas(source.png).map(decodeRgbaPng);
  assert.equal(crops.length, 9);
  crops.forEach((crop, index) => {
    assert.equal(crop.width, 2); assert.equal(crop.height, 5);
    for (let y = 0; y < crop.height; y++) for (let x = 0; x < crop.width; x++) {
      const original = ((Math.floor(index / 3) * crop.height + y) * source.width + (index % 3) * crop.width + x) * 4;
      const offset = (y * crop.width + x) * 4;
      assert.deepEqual(crop.pixels.subarray(offset, offset + 4), source.pixels.subarray(original, original + 4));
    }
    assert.deepEqual(crop.metadata, decodeRgbaPng(source.png).metadata);
  });
});

test('编码不改变像素值与色彩元数据', () => {
  const source = decodeRgbaPng(fixture().png);
  assert.deepEqual(decodeRgbaPng(encodeRgbaPng(source)), source);
});

test('拒绝损坏 CRC 与截断 PNG', () => {
  const { png } = fixture();
  const damaged = Buffer.from(png); damaged[29] ^= 1;
  assert.throws(() => decodeRgbaPng(damaged), /CRC/);
  assert.throws(() => decodeRgbaPng(png.subarray(0, -1)), /截断|越界/);
});

test('拒绝非等分图集与非法行滤波', () => {
  assert.throws(() => splitPngAtlas(fixture({ width: 7 }).png), /被 3 整除/);
  assert.throws(() => decodeRgbaPng(fixture({ badFilter: 5 }).png), /滤波类型无效/);
});

test('拒绝交错、调色板与非八位深度', () => {
  for (const options of [{ interlace: 1 }, { colorType: 3 }, { bitDepth: 16 }]) {
    assert.throws(() => decodeRgbaPng(fixture(options).png), /RGBA8/);
  }
});
