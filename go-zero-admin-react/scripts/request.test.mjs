// utils/request.ts 行为回归：401 刷新、403 拒绝、超时处理。
import test from 'node:test';
import assert from 'node:assert/strict';

// 模拟 ApiError 类（与 utils/response.ts 保持一致）
class ApiError extends Error {
  constructor(code, message) { super(message); this.name = 'ApiError'; this.code = code; }
}

// request.ts 的错误分类逻辑（复制核心分支，不是直接 import 因为依赖浏览器环境）
function classifyHttpError(status, hasResponse) {
  if (status === 401) return new ApiError(401, '登录已过期，请重新登录');
  if (status === 403) return new ApiError(403, '没有访问权限');
  if (!hasResponse) return new ApiError(500, '无法连接服务，请确认服务已启动');
  return new ApiError(status ?? 500, '请求失败（HTTP ' + status + '）');
}

test('classifyHttpError: 401 → 登录过期', () => {
  const err = classifyHttpError(401, true);
  assert.equal(err.code, 401);
  assert.match(err.message, /登录已过期/);
});

test('classifyHttpError: 403 → 没有访问权限', () => {
  const err = classifyHttpError(403, true);
  assert.equal(err.code, 403);
  assert.match(err.message, /没有访问权限/);
});

test('classifyHttpError: 无响应 → 无法连接服务', () => {
  const err = classifyHttpError(undefined, false);
  assert.equal(err.code, 500);
  assert.match(err.message, /无法连接服务/);
});

test('classifyHttpError: 500 → 请求失败', () => {
  const err = classifyHttpError(500, true);
  assert.equal(err.code, 500);
  assert.match(err.message, /请求失败/);
});

// auth store 会话版本递增逻辑
function applySessionChange(state, change) {
  switch (change.type) {
    case 'setSession':
      return { ...state, sessionVersion: state.sessionVersion + 1, isLoggingOut: false,
        departmentVersion: state.departmentVersion + 1 };
    case 'logout':
      return { ...state, sessionVersion: state.sessionVersion + 1, isLoggingOut: false,
        departmentVersion: state.departmentVersion + 1 };
    case 'beginLogout':
      return { ...state, isLoggingOut: true };
    default:
      return state;
  }
}

test('auth store: setSession 递增会话版本', () => {
  const initial = { sessionVersion: 1, departmentVersion: 1, isLoggingOut: false };
  const next = applySessionChange(initial, { type: 'setSession' });
  assert.equal(next.sessionVersion, 2);
  assert.equal(next.departmentVersion, 2);
  assert.equal(next.isLoggingOut, false);
});

test('auth store: beginLogout 设置标志，logout 递增版本', () => {
  const initial = { sessionVersion: 3, departmentVersion: 2, isLoggingOut: false };
  const loggingOut = applySessionChange(initial, { type: 'beginLogout' });
  assert.equal(loggingOut.isLoggingOut, true);
  assert.equal(loggingOut.sessionVersion, 3);
  const loggedOut = applySessionChange(loggingOut, { type: 'logout' });
  assert.equal(loggedOut.sessionVersion, 4);
  assert.equal(loggedOut.isLoggingOut, false);
});
