import { useCallback, useLayoutEffect, useRef, useState } from 'react';
import type { KeyboardEvent, TouchEvent, WheelEvent } from 'react';

export interface ChatScrollPosition {
  following: boolean;
  top: number;
}

interface ChatScrollOptions {
  sending: boolean;
  firstMessageId?: number;
  position?: { current: ChatScrollPosition | null };
}

export function createChatScrollController(
  viewport: HTMLDivElement, content: HTMLDivElement,
  onShowLatestChange: (shown: boolean) => void, requestHistoryCommit: () => void,
  initialPosition?: ChatScrollPosition | null,
) {
  let following = initialPosition?.following ?? true;
  let disposed = false;
  let frame: number | null = null;
  let lastTop = viewport.scrollTop;
  let lastHeight = viewport.scrollHeight;
  let lastViewportHeight = viewport.clientHeight;
  let showLatest: boolean | null = null;
  let history: { height: number; top: number } | null = null;
  let touchY: number | null = null;
  let userPauseTop: number | null = null;

  const distanceToLatest = () => Math.max(0, viewport.scrollHeight - viewport.clientHeight - viewport.scrollTop);
  function updateLatestAction() {
    const shown = !following && distanceToLatest() > 1;
    if (shown !== showLatest) {
      showLatest = shown;
      onShowLatestChange(shown);
    }
  }
  function cancelFrame() {
    if (frame !== null) cancelAnimationFrame(frame);
    frame = null;
  }
  function rememberPosition() {
    lastTop = viewport.scrollTop;
    lastHeight = viewport.scrollHeight;
    lastViewportHeight = viewport.clientHeight;
  }
  function writeLatest() {
    viewport.scrollTop = viewport.scrollHeight;
    rememberPosition();
    updateLatestAction();
  }
  function scheduleLatest() {
    if (disposed || !following || history || frame !== null) return;
    frame = requestAnimationFrame(() => {
      frame = null;
      if (!disposed && following && !history) writeLatest();
    });
  }
  function pauseForUser() {
    if (disposed) return;
    following = false;
    userPauseTop = viewport.scrollTop;
    cancelFrame();
    updateLatestAction();
  }
  const observer = new ResizeObserver(() => {
    if (disposed) return;
    updateLatestAction();
    scheduleLatest();
  });
  // 同时观察内容和视口，覆盖流式换行、工具展开、输入区伸缩及窗口调整。
  observer.observe(content);
  observer.observe(viewport);
  if (initialPosition && !following) {
    viewport.scrollTop = initialPosition.top;
    rememberPosition();
    userPauseTop = viewport.scrollTop;
  }
  updateLatestAction();
  scheduleLatest();

  return {
    onScroll() {
      if (disposed) return;
      const top = viewport.scrollTop;
      const layoutChanged = viewport.scrollHeight !== lastHeight || viewport.clientHeight !== lastViewportHeight;
      if (history) {
        history.top = top;
        if (userPauseTop !== null && Math.abs(top - userPauseTop) > 1) userPauseTop = null;
      } else if (userPauseTop !== null) {
        // 用户输入可能先于已排队的程序scroll；同位置事件不能撤销上翻意图。
        if (Math.abs(top - userPauseTop) > 1) {
          following = top > userPauseTop && distanceToLatest() <= 64;
          userPauseTop = null;
        }
      } else if (distanceToLatest() <= 1) {
        following = true;
      } else if (following && layoutChanged) {
        // 多次布局提交可能先夹小scrollTop再缩小视口，保持原有跟随意图。
        following = true;
      } else if (top < lastTop - 1) {
        following = false;
      } else if (top > lastTop && distanceToLatest() <= 64) {
        following = true;
      } else if (top > lastTop + 1) {
        following = false;
      }
      rememberPosition();
      if (!following) cancelFrame();
      updateLatestAction();
      scheduleLatest();
    },
    onWheel(deltaY: number, ctrlKey: boolean) {
      if (deltaY < 0 && !ctrlKey) pauseForUser();
    },
    onKeyDown(key: string, shiftKey: boolean) {
      if (key === 'ArrowUp' || key === 'PageUp' || key === 'Home' || (key === ' ' && shiftKey)) pauseForUser();
    },
    onTouchStart(y: number | null) {
      if (!disposed) touchY = y;
    },
    onTouchMove(y: number | null) {
      if (disposed) return;
      if (y !== null && touchY !== null && y > touchY) pauseForUser();
      touchY = y;
    },
    scrollToLatest() {
      if (disposed) return;
      history = null;
      userPauseTop = null;
      following = true;
      writeLatest();
      scheduleLatest();
    },
    prepareForHistory() {
      if (disposed) return;
      history = { height: viewport.scrollHeight, top: viewport.scrollTop };
      following = false;
      cancelFrame();
      updateLatestAction();
    },
    restoreHistory() {
      if (disposed || !history) return;
      const anchor = history;
      history = null;
      viewport.scrollTop = anchor.top + viewport.scrollHeight - anchor.height;
      rememberPosition();
      if (userPauseTop !== null) userPauseTop = viewport.scrollTop;
      else if (distanceToLatest() <= 1) following = true;
      updateLatestAction();
    },
    finishHistory() {
      if (disposed || !history) return;
      requestHistoryCommit();
    },
    getPosition(): ChatScrollPosition {
      return { following, top: viewport.scrollTop };
    },
    dispose() {
      disposed = true;
      cancelFrame();
      observer.disconnect();
    },
  };
}

export function useChatScroll({ sending, firstMessageId, position }: ChatScrollOptions) {
  const viewportRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const controller = useRef<ReturnType<typeof createChatScrollController> | null>(null);
  const wasSending = useRef(sending);
  const [showLatest, setShowLatest] = useState(false);
  const [historyRevision, setHistoryRevision] = useState(0);

  useLayoutEffect(() => {
    if (!viewportRef.current || !contentRef.current) return;
    const active = createChatScrollController(viewportRef.current, contentRef.current, setShowLatest, () => {
      // 失败或空页不改变首条 ID，仍需等本次提交完成后释放历史锚点。
      setHistoryRevision(revision => revision + 1);
    }, position?.current);
    controller.current = active;
    return () => {
      if (position) position.current = active.getPosition();
      active.dispose();
      controller.current = null;
    };
  }, [position]);
  useLayoutEffect(() => { controller.current?.restoreHistory(); }, [firstMessageId, historyRevision]);
  useLayoutEffect(() => {
    if (sending && !wasSending.current) controller.current?.scrollToLatest();
    wasSending.current = sending;
  }, [sending]);

  const onScroll = useCallback(() => controller.current?.onScroll(), []);
  const onWheel = useCallback((event: WheelEvent<HTMLDivElement>) => {
    if (!event.defaultPrevented) controller.current?.onWheel(event.deltaY, event.ctrlKey);
  }, []);
  const onKeyDown = useCallback((event: KeyboardEvent<HTMLDivElement>) => {
    const target = event.target;
    if (event.defaultPrevented || (target instanceof HTMLElement && (target.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName)))) return;
    controller.current?.onKeyDown(event.key, event.shiftKey);
  }, []);
  const onTouchStart = useCallback((event: TouchEvent<HTMLDivElement>) => {
    controller.current?.onTouchStart(event.touches.length === 1 ? event.touches[0].clientY : null);
  }, []);
  const onTouchMove = useCallback((event: TouchEvent<HTMLDivElement>) => {
    controller.current?.onTouchMove(event.touches.length === 1 ? event.touches[0].clientY : null);
  }, []);
  const scrollToLatest = useCallback(() => controller.current?.scrollToLatest(), []);
  const prepareForHistory = useCallback(() => controller.current?.prepareForHistory(), []);
  const finishHistory = useCallback(() => controller.current?.finishHistory(), []);
  return { viewportRef, contentRef, onScroll, onWheel, onKeyDown, onTouchStart, onTouchMove, showLatest, scrollToLatest, prepareForHistory, finishHistory };
}
