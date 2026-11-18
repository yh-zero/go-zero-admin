import type { AiChatMessage } from '../../api/types';

export interface ChatKnowledgeSelection { kb_id: number; doc_id?: number; grounded: boolean }
export const automaticKnowledge: ChatKnowledgeSelection = { kb_id: 0, grounded: false };

// 页内最新消息记录最近一次请求使用的范围；旧消息中某篇文档的引用并不代表选中了它。
export function latestConversationKnowledge(messages: readonly AiChatMessage[]): ChatKnowledgeSelection {
  const last = messages.reduce<AiChatMessage | undefined>((latest, message) => !latest || message.id > latest.id ? message : latest, undefined);
  if (!last?.kb_id || last.kb_id <= 0) return automaticKnowledge;
  if (last.doc_id && last.doc_id > 0) return { kb_id: last.kb_id, doc_id: last.doc_id, grounded: true };
  return { kb_id: last.kb_id, grounded: last.grounded === true };
}
interface StoredSelection extends ChatKnowledgeSelection { entryKey?: string }
type SessionSelections = Record<string, StoredSelection>;
const fallback = new Map<string, SessionSelections>();
const storageKey = (sessionId: string) => `ai-chat-knowledge:${sessionId}`;
const choiceKey = (departmentId: number | null, conversationId: number) => JSON.stringify([departmentId, conversationId]);

function sessionSelections(sessionId: string): SessionSelections {
  const cached = fallback.get(sessionId);
  if (cached) return cached;
  try {
    const saved: unknown = JSON.parse(sessionStorage.getItem(storageKey(sessionId)) ?? '{}');
    return saved && typeof saved === 'object' && !Array.isArray(saved) ? saved as SessionSelections : {};
  } catch { return {}; }
}

export function readChatKnowledge(sessionId: string, departmentId: number | null, conversationId: number): StoredSelection | undefined {
  const saved = sessionSelections(sessionId)[choiceKey(departmentId, conversationId)];
  if (!saved || !Number.isSafeInteger(saved.kb_id) || saved.kb_id < 0 || typeof saved.grounded !== 'boolean') return undefined;
  if (saved.doc_id !== undefined && (!Number.isSafeInteger(saved.doc_id) || saved.doc_id <= 0 || saved.kb_id <= 0 || !saved.grounded)) return undefined;
  return { kb_id: saved.kb_id, ...(saved.doc_id ? { doc_id: saved.doc_id } : {}), grounded: saved.kb_id > 0 && saved.grounded, entryKey: typeof saved.entryKey === 'string' ? saved.entryKey : undefined };
}

export function writeChatKnowledge(sessionId: string, departmentId: number | null, conversationId: number, selection?: ChatKnowledgeSelection, entryKey?: string) {
  if (!sessionId) return;
  const choices = sessionSelections(sessionId);
  const key = choiceKey(departmentId, conversationId);
  if (selection) choices[key] = { kb_id: selection.kb_id, doc_id: selection.doc_id, grounded: selection.grounded, entryKey };
  else delete choices[key];
  try { sessionStorage.setItem(storageKey(sessionId), JSON.stringify(choices)); fallback.delete(sessionId); }
  catch { fallback.set(sessionId, choices); }
}

export function clearChatKnowledgeSession(sessionId: string) {
  if (!sessionId) return;
  fallback.delete(sessionId);
  try { sessionStorage.removeItem(storageKey(sessionId)); } catch { /* 已退出的登录会话不会再读取此键。 */ }
}
