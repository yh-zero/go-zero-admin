import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';

test('AI聊天滚动跟随与历史视口', async t => {
  const original = {
    requestAnimationFrame: globalThis.requestAnimationFrame,
    cancelAnimationFrame: globalThis.cancelAnimationFrame,
    ResizeObserver: globalThis.ResizeObserver,
  };
  const frames = new Map();
  const observers = new Set();
  let nextFrame = 0;
  globalThis.requestAnimationFrame = callback => { const id = ++nextFrame; frames.set(id, callback); return id; };
  globalThis.cancelAnimationFrame = id => { frames.delete(id); };
  globalThis.ResizeObserver = class {
    targets = new Set();
    constructor(callback) { this.callback = callback; observers.add(this); }
    observe(target) { this.targets.add(target); }
    disconnect() { this.targets.clear(); observers.delete(this); }
  };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const { createChatScrollController } = await server.ssrLoadModule('/src/pages/ai/useChatScroll.ts');
    const flush = () => {
      const pending = [...frames.values()];
      frames.clear();
      for (const callback of pending) callback(0);
    };
    const fixture = initialPosition => {
      let top = 0;
      const viewport = {
        scrollHeight: 1000, clientHeight: 400,
        get scrollTop() { return top; },
        set scrollTop(value) { top = Math.max(0, Math.min(value, this.scrollHeight - this.clientHeight)); },
      };
      const content = {};
      const shown = [];
      const historyCommits = [];
      const controller = createChatScrollController(viewport, content, value => shown.push(value), () => {
        historyCommits.push(() => controller.restoreHistory());
      }, initialPosition);
      const commitHistory = () => { for (const callback of historyCommits.splice(0)) callback(); };
      const resize = target => {
        for (const observer of observers) {
          if (observer.targets.has(target)) observer.callback([{ target }], observer);
        }
      };
      return { viewport, content, shown, controller, resize, commitHistory, historyCommits };
    };

    await t.test('初载滚至底部，文本或工具内容增长在一帧中合并跟随', () => {
      const { viewport, content, controller, resize } = fixture();
      try {
        flush();
        assert.equal(viewport.scrollTop, 600);
        viewport.scrollHeight = 1300;
        resize(content);
        resize(content);
        resize(content);
        assert.equal(frames.size, 1);
        flush();
        assert.equal(viewport.scrollTop, 900);
      } finally { controller.dispose(); }
    });

    await t.test('轻微上翻立即暂停，后续增量与视口高度变化保留阅读位置', () => {
      const { viewport, content, shown, controller, resize } = fixture();
      try {
        flush();
        viewport.scrollTop = 580;
        controller.onScroll();
        assert.equal(shown.at(-1), true);
        viewport.scrollHeight = 1100;
        resize(content);
        flush();
        assert.equal(viewport.scrollTop, 580);
        viewport.clientHeight = 500;
        resize(viewport);
        flush();
        assert.equal(viewport.scrollTop, 580);
        assert.equal(shown.at(-1), true);
      } finally { controller.dispose(); }
    });

    await t.test('手动回到最新区域后恢复增量跟随', () => {
      const { viewport, content, shown, controller, resize } = fixture();
      try {
        flush();
        viewport.scrollTop = 300;
        controller.onScroll();
        viewport.scrollHeight = 1200;
        resize(content);
        flush();
        viewport.scrollTop = 760;
        controller.onScroll();
        flush();
        assert.equal(shown.at(-1), false);
        assert.equal(viewport.scrollTop, 800);
        viewport.scrollHeight = 1500;
        resize(content);
        flush();
        assert.equal(viewport.scrollTop, 1100);
      } finally { controller.dispose(); }
    });

    await t.test('同一位置的重复滚动事件不能恢复已暂停的跟随', () => {
      const { viewport, content, shown, controller, resize } = fixture();
      try {
        flush();
        viewport.scrollTop = 580;
        controller.onScroll();
        controller.onScroll();
        viewport.scrollHeight = 1200;
        resize(content);
        flush();
        assert.equal(viewport.scrollTop, 580);
        assert.equal(shown.at(-1), true);
      } finally { controller.dispose(); }
    });

    await t.test('点击最新恢复跟随并丢弃尚未完成的历史锚点', () => {
      const { viewport, content, shown, controller, resize } = fixture();
      try {
        flush();
        viewport.scrollTop = 100;
        controller.onScroll();
        controller.prepareForHistory();
        controller.scrollToLatest();
        flush();
        assert.equal(viewport.scrollTop, 600);
        assert.equal(shown.at(-1), false);
        viewport.scrollHeight = 1300;
        controller.restoreHistory();
        resize(content);
        flush();
        assert.equal(viewport.scrollTop, 900);
      } finally { controller.dispose(); }
    });

    await t.test('前插历史消息按新增高度补偿并继续停留在历史位置', () => {
      const { viewport, content, controller, resize } = fixture();
      try {
        flush();
        viewport.scrollTop = 140;
        controller.onScroll();
        controller.prepareForHistory();
        viewport.scrollHeight = 1300;
        resize(content);
        flush();
        assert.equal(viewport.scrollTop, 140);
        controller.restoreHistory();
        assert.equal(viewport.scrollTop, 440);
        viewport.scrollHeight = 1400;
        resize(content);
        flush();
        assert.equal(viewport.scrollTop, 440);
      } finally { controller.dispose(); }
    });

    await t.test('等待历史请求期间继续滚动，以用户当前阅读位置补偿', () => {
      const { viewport, controller } = fixture();
      try {
        flush();
        viewport.scrollTop = 140;
        controller.onScroll();
        controller.prepareForHistory();
        viewport.scrollTop = 200;
        controller.onScroll();
        viewport.scrollHeight = 1300;
        controller.restoreHistory();
        assert.equal(viewport.scrollTop, 500);
      } finally { controller.dispose(); }
    });

    await t.test('历史请求失败或返回空页后，手动到底恢复跟随', () => {
      const { viewport, content, controller, resize, commitHistory } = fixture();
      try {
        flush();
        viewport.scrollTop = 140;
        controller.onScroll();
        controller.prepareForHistory();
        // 失败与空页均不改变首条消息 ID，但请求结束仍需释放历史锚点。
        controller.finishHistory();
        commitHistory();
        assert.equal(viewport.scrollTop, 140);
        viewport.scrollTop = 600;
        controller.onScroll();
        viewport.scrollHeight = 1200;
        resize(content);
        flush();
        assert.equal(viewport.scrollTop, 800);
      } finally { controller.dispose(); }
    });

    await t.test('历史结束通知等待提交完成后再读取前插后的内容高度', () => {
      const { viewport, controller, commitHistory } = fixture();
      try {
        flush();
        viewport.scrollTop = 140;
        controller.onScroll();
        controller.prepareForHistory();
        controller.finishHistory();
        assert.equal(viewport.scrollTop, 140);
        viewport.scrollHeight = 1300;
        commitHistory();
        assert.equal(viewport.scrollTop, 440);
      } finally { controller.dispose(); }
    });

    await t.test('等待历史期间已回到底部，结束后无需额外滚动即可继续跟随', () => {
      const { viewport, content, controller, resize, commitHistory } = fixture();
      try {
        flush();
        viewport.scrollTop = 140;
        controller.onScroll();
        controller.prepareForHistory();
        viewport.scrollTop = 600;
        controller.onScroll();
        controller.finishHistory();
        commitHistory();
        viewport.scrollHeight = 1200;
        resize(content);
        flush();
        assert.equal(viewport.scrollTop, 800);
      } finally { controller.dispose(); }
    });

    await t.test('卸载后的迟到历史结束通知不触发提交', () => {
      const { viewport, controller, historyCommits } = fixture();
      flush();
      viewport.scrollTop = 140;
      controller.onScroll();
      controller.prepareForHistory();
      controller.dispose();
      controller.finishHistory();
      assert.equal(historyCommits.length, 0);
      assert.equal(viewport.scrollTop, 140);
    });

    await t.test('跟随期间视口收缩和内容重排仍保持最新消息可见', () => {
      const { viewport, controller, resize } = fixture();
      try {
        flush();
        viewport.clientHeight = 300;
        resize(viewport);
        flush();
        assert.equal(viewport.scrollTop, 700);
        viewport.scrollHeight = 1600;
        resize(viewport);
        flush();
        assert.equal(viewport.scrollTop, 1300);
      } finally { controller.dispose(); }
    });

    await t.test('布局先扩后缩引起的自动clamp不被当作手动上翻', () => {
      for (const observerFirst of [true, false]) {
        const { viewport, content, shown, controller, resize } = fixture();
        try {
          flush();
          // pending移除先夹至116；新消息与反馈提交后最大位置变成564。
          viewport.scrollHeight = 516;
          viewport.scrollTop = Math.min(viewport.scrollTop, viewport.scrollHeight - viewport.clientHeight);
          assert.equal(viewport.scrollTop, 116);
          viewport.scrollHeight = 584;
          viewport.clientHeight = 20;
          if (observerFirst) resize(content);
          controller.onScroll();
          if (!observerFirst) resize(viewport);
          flush();
          assert.equal(viewport.scrollTop, 564);
          assert.notEqual(shown.at(-1), true);
          viewport.clientHeight = 200;
          resize(viewport);
          flush();
          assert.equal(viewport.scrollTop, 384);
        } finally { controller.dispose(); }
      }
    });

    await t.test('布局变化同时按PageUp或滚轮上翻仍立即暂停并取消跟随帧', () => {
      for (const intent of [controller => controller.onKeyDown('PageUp', false), controller => controller.onWheel(-20, false)]) {
        const { viewport, content, shown, controller, resize } = fixture();
        try {
          flush();
          viewport.scrollHeight = 1200;
          resize(content);
          assert.equal(frames.size, 1);
          intent(controller);
          viewport.scrollTop = 500;
          controller.onScroll();
          flush();
          assert.equal(viewport.scrollTop, 500);
          assert.equal(shown.at(-1), true);
          viewport.scrollHeight = 1400;
          resize(content);
          flush();
          assert.equal(viewport.scrollTop, 500);
        } finally { controller.dispose(); }
      }
    });

    await t.test('PageUp后的同位置程序滚动不能覆盖用户暂停意图', () => {
      const { viewport, content, shown, controller, resize } = fixture();
      try {
        flush();
        controller.onKeyDown('PageUp', false);
        controller.onScroll();
        assert.deepEqual(controller.getPosition(), { following: false, top: 600 });
        viewport.scrollHeight = 1200;
        resize(content);
        viewport.scrollTop = 500;
        controller.onScroll();
        flush();
        assert.equal(viewport.scrollTop, 500);
        assert.equal(shown.at(-1), true);
        viewport.scrollTop = 780;
        controller.onScroll();
        flush();
        assert.equal(viewport.scrollTop, 800);
        assert.deepEqual(controller.getPosition(), { following: true, top: 800 });
      } finally { controller.dispose(); }
    });

    await t.test('单指向下滑动暂停上翻，多指缩放不改变跟随', () => {
      const { viewport, content, shown, controller, resize } = fixture();
      try {
        flush();
        controller.onTouchStart(100);
        viewport.scrollHeight = 1200;
        resize(content);
        controller.onTouchMove(150);
        viewport.scrollTop = 500;
        controller.onScroll();
        flush();
        assert.equal(viewport.scrollTop, 500);
        assert.equal(shown.at(-1), true);
        controller.scrollToLatest();
        flush();
        controller.onTouchStart(100);
        controller.onTouchMove(null);
        controller.onTouchMove(150);
        viewport.scrollHeight = 1400;
        resize(content);
        flush();
        assert.equal(viewport.scrollTop, 1000);
      } finally { controller.dispose(); }
    });

    await t.test('缩放滚轮、向下滚动与其他按键不暂停自动跟随', () => {
      const { viewport, content, controller, resize } = fixture();
      try {
        flush();
        controller.onWheel(-20, true);
        controller.onWheel(20, false);
        controller.onKeyDown('PageDown', false);
        controller.onKeyDown(' ', false);
        viewport.scrollHeight = 1200;
        resize(content);
        flush();
        assert.equal(viewport.scrollTop, 800);
      } finally { controller.dispose(); }
    });

    await t.test('暂停位置快照保留跟随意图和实际阅读位置', () => {
      const { viewport, controller } = fixture();
      try {
        flush();
        controller.onKeyDown('PageUp', false);
        viewport.scrollTop = 250;
        controller.onScroll();
        const position = controller.getPosition();
        assert.deepEqual(position, { following: false, top: 250 });
        controller.scrollToLatest();
        assert.deepEqual(controller.getPosition(), { following: true, top: 600 });
        assert.deepEqual(position, { following: false, top: 250 }, '已保存快照不随旧控制器后续变化');
      } finally { controller.dispose(); }
    });

    await t.test('新建转正式会话时恢复暂停位置，后续内容变化仍保留阅读位置', () => {
      const { viewport, content, shown, controller, resize } = fixture({ following: false, top: 250 });
      try {
        flush();
        assert.equal(viewport.scrollTop, 250);
        assert.equal(shown.at(-1), true);
        viewport.scrollHeight = 1400;
        resize(content);
        flush();
        assert.equal(viewport.scrollTop, 250);
        assert.deepEqual(controller.getPosition(), { following: false, top: 250 });
      } finally { controller.dispose(); }
    });

    await t.test('暂停快照恢复到底部时，初始化程序滚动不会恢复跟随', () => {
      const { viewport, content, controller, resize } = fixture({ following: false, top: 600 });
      try {
        controller.onScroll();
        assert.deepEqual(controller.getPosition(), { following: false, top: 600 });
        viewport.scrollHeight = 1200;
        resize(content);
        flush();
        assert.equal(viewport.scrollTop, 600);
        controller.scrollToLatest();
        controller.onScroll();
        flush();
        assert.equal(viewport.scrollTop, 800);
        assert.deepEqual(controller.getPosition(), { following: true, top: 800 });
      } finally { controller.dispose(); }
    });

    await t.test('恢复跟随中的快照时采用新内容的底部而非旧top', () => {
      const { viewport, content, shown, controller, resize } = fixture({ following: true, top: 250 });
      try {
        flush();
        assert.equal(viewport.scrollTop, 600);
        assert.equal(shown.at(-1), false, '替换位置容器后清除旧控制器的回到最新提示');
        assert.deepEqual(controller.getPosition(), { following: true, top: 600 });
        viewport.scrollHeight = 1400;
        resize(content);
        flush();
        assert.equal(viewport.scrollTop, 1000);
      } finally { controller.dispose(); }
    });

    await t.test('卸载清理观察器和帧，迟到回调不再滚动或更新界面状态', () => {
      const { viewport, content, shown, controller, resize } = fixture();
      flush();
      const observer = [...observers].find(value => value.targets.has(content));
      assert.ok(observer);
      viewport.scrollHeight = 1300;
      resize(content);
      const lateFrame = [...frames.values()][0];
      assert.equal(typeof lateFrame, 'function');
      controller.dispose();
      assert.equal(frames.size, 0);
      assert.equal(observers.has(observer), false);
      const count = shown.length;
      lateFrame(0);
      observer.callback([{ target: content }], observer);
      controller.scrollToLatest();
      controller.onScroll();
      assert.equal(viewport.scrollTop, 600);
      assert.equal(shown.length, count);
      assert.equal(frames.size, 0);
    });
  } finally {
    await server.close();
    Object.assign(globalThis, original);
  }
});
