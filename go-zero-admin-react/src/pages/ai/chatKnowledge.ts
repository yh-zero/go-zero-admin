import type { AiChatMessage, AiSendMessageReq } from '../../api/types';

export interface ChatKnowledgeSelection { kb_id: number; doc_id?: number; grounded: boolean }
export interface StoredChatKnowledge extends ChatKnowledgeSelection { entryKey?: string; confirmedMessageId?: number; pending?: true }
export const automaticKnowledge: ChatKnowledgeSelection = { kb_id: 0, grounded: false };

// 页内最新消息记录最近一次请求使用的范围；旧消息中某篇文档的引用并不代表选中了它。
export function latestConversationKnowledge(messages: readonly AiChatMessage[], local?: StoredChatKnowledge): ChatKnowledgeSelection {
  const last = messages.reduce<AiChatMessage | undefined>((latest, message) => !latest || message.id > latest.id ? message : latest, undefined);
  // 无确认标记的旧记录也可能是未发送的选择，不能扩大用户明确指定的范围。
  if (local && (!local.confirmedMessageId || !last || last.id <= local.confirmedMessageId)) return local;
  if (!last?.kb_id || last.kb_id <= 0) return automaticKnowledge;
  if (last.doc_id && last.doc_id > 0) return { kb_id: last.kb_id, doc_id: last.doc_id, grounded: true };
  return { kb_id: last.kb_id, grounded: last.grounded === true };
}
function sameKnowledge(a: ChatKnowledgeSelection, b: ChatKnowledgeSelection) {
  return a.kb_id === b.kb_id && a.doc_id === b.doc_id && a.grounded === b.grounded;
}

export function hasLegacyKnowledgeConflict(messages: readonly AiChatMessage[], local?: StoredChatKnowledge) {
  return !!local && !local.pending && !local.confirmedMessageId && messages.length > 0 && !sameKnowledge(local, latestConversationKnowledge(messages));
}
type SessionSelections = Record<string, StoredChatKnowledge>;
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

export function readChatKnowledge(sessionId: string, departmentId: number | null, conversationId: number): StoredChatKnowledge | undefined {
  const saved = sessionSelections(sessionId)[choiceKey(departmentId, conversationId)];
  if (!saved || !Number.isSafeInteger(saved.kb_id) || saved.kb_id < 0 || typeof saved.grounded !== 'boolean') return undefined;
  if (saved.doc_id !== undefined && (!Number.isSafeInteger(saved.doc_id) || saved.doc_id <= 0 || saved.kb_id <= 0 || !saved.grounded)) return undefined;
  const confirmedMessageId = saved.confirmedMessageId;
  return { kb_id: saved.kb_id, ...(saved.doc_id ? { doc_id: saved.doc_id } : {}), grounded: saved.kb_id > 0 && saved.grounded,
    ...(typeof confirmedMessageId === 'number' && Number.isSafeInteger(confirmedMessageId) && confirmedMessageId > 0 ? { confirmedMessageId } : {}),
    ...(saved.pending === true ? { pending: true } : {}),
    entryKey: typeof saved.entryKey === 'string' ? saved.entryKey : undefined };
}

export function writeChatKnowledge(sessionId: string, departmentId: number | null, conversationId: number, selection?: StoredChatKnowledge, entryKey?: string) {
  if (!sessionId) return;
  const choices = sessionSelections(sessionId);
  const key = choiceKey(departmentId, conversationId);
  if (selection) choices[key] = { kb_id: selection.kb_id, doc_id: selection.doc_id, grounded: selection.grounded,
    confirmedMessageId: selection.confirmedMessageId, pending: conversationId > 0 && !selection.confirmedMessageId ? true : undefined, entryKey };
  else delete choices[key];
  try { sessionStorage.setItem(storageKey(sessionId), JSON.stringify(choices)); fallback.delete(sessionId); }
  catch { fallback.set(sessionId, choices); }
}

export function confirmChatKnowledge(sessionId: string, departmentId: number | null, conversationId: number,
  request: Pick<AiSendMessageReq, 'kb_id' | 'doc_id' | 'grounded'>, messageId: number, preserveNewDraft = false) {
  const saved = readChatKnowledge(sessionId, departmentId, conversationId);
  const kb_id = request.kb_id ?? 0, doc_id = request.doc_id || undefined, grounded = !!request.grounded;
  if (saved?.confirmedMessageId && saved.confirmedMessageId > messageId) return;
  // 重查确认的是原请求；用户在等待期间另选的下一问范围继续作为草稿。
  if (saved && !saved.confirmedMessageId && (preserveNewDraft || !sameKnowledge(saved, { kb_id, doc_id, grounded }))) return;
  writeChatKnowledge(sessionId, departmentId, conversationId, { kb_id, doc_id, grounded, confirmedMessageId: messageId });
}

export function clearChatKnowledgeSession(sessionId: string) {
  if (!sessionId) return;
  fallback.delete(sessionId);
  try { sessionStorage.removeItem(storageKey(sessionId)); } catch { /* 已退出的登录会话不会再读取此键。 */ }
}
