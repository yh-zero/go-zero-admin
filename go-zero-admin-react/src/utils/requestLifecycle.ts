const pending = new Map<AbortSignal, AbortController>();
export function trackRequest(existing?: AbortSignal): AbortSignal {
  const controller = new AbortController();
  const signal = existing ? AbortSignal.any([existing, controller.signal]) : controller.signal;
  pending.set(signal, controller);
  return signal;
}
export function releaseRequest(signal: unknown) { pending.delete(signal as AbortSignal); }
export function cancelPendingRequests() {
  for (const controller of pending.values()) controller.abort();
  pending.clear();
}
