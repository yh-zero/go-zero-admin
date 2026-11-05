import { useCallback, useEffect, useRef, useState } from 'react';
import type { KeyboardEvent, PointerEvent } from 'react';
import { clampSidebarWidth, SIDEBAR_DEFAULT_WIDTH, SIDEBAR_MAX_WIDTH, SIDEBAR_MIN_WIDTH, useLayoutStore } from '../stores/layout';

export function useSidebarResize(collapsed: boolean) {
  const savedWidth = useLayoutStore(state => state.sidebarWidth);
  const saveWidth = useLayoutStore(state => state.setSidebarWidth);
  const [draftWidth, setDraftWidth] = useState<number | null>(null);
  const handle = useRef<HTMLDivElement>(null);
  const drag = useRef<{ pointerId: number; startX: number; startWidth: number } | null>(null);
  const width = draftWidth ?? savedWidth;
  const dragging = draftWidth !== null;

  const cancelDrag = useCallback(() => {
    const pointerId = drag.current?.pointerId;
    drag.current = null;
    setDraftWidth(null);
    if (pointerId !== undefined && handle.current?.hasPointerCapture(pointerId)) {
      handle.current.releasePointerCapture(pointerId);
    }
  }, []);
  // collapsed 变化时若正在拖拽则取消：派生自渲染而非 effect，避免级联渲染。
  // 在 startDrag/moveDrag 入口判断 collapsed 也能达到同样效果，且更符合"事件驱动"原则。
  useEffect(() => {
    if (!dragging) return;
    const { cursor, userSelect } = document.body.style;
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
    return () => {
      document.body.style.cursor = cursor;
      document.body.style.userSelect = userSelect;
    };
  }, [dragging]);

  const startDrag = (event: PointerEvent<HTMLDivElement>) => {
    if (collapsed) return; // 折叠状态下不启动拖拽
    if (!event.isPrimary || event.button !== 0) return;
    event.preventDefault();
    event.currentTarget.focus({ preventScroll: true });
    event.currentTarget.setPointerCapture(event.pointerId);
    drag.current = { pointerId: event.pointerId, startX: event.clientX, startWidth: width };
    setDraftWidth(width);
  };
  const moveDrag = (event: PointerEvent<HTMLDivElement>) => {
    const active = drag.current;
    if (collapsed) { if (active) cancelDrag(); return; } // 折叠时取消进行中的拖拽
    if (active?.pointerId === event.pointerId) {
      setDraftWidth(clampSidebarWidth(active.startWidth + event.clientX - active.startX));
    }
  };
  const endDrag = (event: PointerEvent<HTMLDivElement>) => {
    const active = drag.current;
    if (active?.pointerId !== event.pointerId) return;
    saveWidth(active.startWidth + event.clientX - active.startX);
    cancelDrag();
  };
  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Escape' && drag.current) {
      event.preventDefault();
      cancelDrag();
      return;
    }
    if (event.altKey || event.ctrlKey || event.metaKey) return;
    const step = event.shiftKey ? 40 : 10;
    const nextWidth = { ArrowLeft: width - step, ArrowRight: width + step, Home: SIDEBAR_MIN_WIDTH, End: SIDEBAR_MAX_WIDTH }[event.key];
    if (nextWidth === undefined || dragging) return;
    event.preventDefault();
    saveWidth(nextWidth);
  };

  return { width, dragging, handleProps: {
    ref: handle,
    onPointerDown: startDrag, onPointerMove: moveDrag, onPointerUp: endDrag,
    onPointerCancel: cancelDrag, onLostPointerCapture: cancelDrag, onKeyDown,
    onDoubleClick: () => saveWidth(SIDEBAR_DEFAULT_WIDTH),
  } };
}
