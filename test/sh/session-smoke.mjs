// Session regression against a local development API; no SQL or JWT fabrication.
// node test/sh/session-smoke.mjs [admin-session.json]
// Or set GOZERO_TEST_TOKEN. All writes target unique session_smoke_* fixtures.
// GOZERO_SESSION_SCENARIOS=default_role,reset runs only selected scenarios and
// writes a separate report. CAPTCHA mistakes retry with a new image, at most 3 attempts.
// CAPTCHA is always requested from the real endpoint and entered manually.
// Each prompt writes a PNG under the OS temporary directory. Type skip to skip
// the current scenario; skipped checks are explicitly recorded in the report.
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { createInterface } from "node:readline/promises";
import { randomBytes } from "node:crypto";

const base = process.env.GOZERO_TEST_URL || "http://127.0.0.1:7001";
if (!/^http:\/\/(127\.0\.0\.1|localhost):\d+$/.test(base)) throw Error("Only local development APIs are supported");
const adminToken = process.env.GOZERO_TEST_TOKEN ||
  (process.argv[2] && JSON.parse(fs.readFileSync(process.argv[2], "utf8")).accessToken);
if (!adminToken) throw Error("Provide an authenticated admin session file or GOZERO_TEST_TOKEN");
const allScenarios = ["profile", "password", "freeze", "default_role", "linked_role", "reset", "logout", "delete"];
const requestedScenarios = process.env.GOZERO_SESSION_SCENARIOS?.trim();
const selectedScenarios = requestedScenarios
  ? new Set(requestedScenarios.split(",").map((name) => name.trim()).filter(Boolean))
  : null;
if (selectedScenarios && (selectedScenarios.size === 0 || [...selectedScenarios].some((name) => !allScenarios.includes(name))))
  throw Error("GOZERO_SESSION_SCENARIOS accepts only: " + allScenarios.join(","));
const prefix = "session_smoke_" + Date.now() + "_" + randomBytes(2).toString("hex");
const folder = path.join(os.tmpdir(), "go-zero-admin-regression", prefix);
fs.mkdirSync(folder, { recursive: true });
const input = createInterface({ input: process.stdin, output: process.stdout });
const results = [];
const cleanups = [];
const roles = [Math.floor(Date.now()/1000)+100000, Math.floor(Date.now()/1000)+100001];
let sequence = 0;
class Skip extends Error {}
async function request(method, route, body, token = adminToken) {
  const response = await fetch(base + "/v1/sys" + route, {
    method,
    headers: { "Content-Type": "application/json", ...(token ? { Authorization: "Bearer " + token } : {}) },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(15000),
  });
  return { status: response.status, data: await response.json().catch(() => ({})) };
}
async function ok(method, route, body, token = adminToken) {
  const response = await request(method, route, body, token);
  assert.equal(response.status, 200, route + " HTTP " + response.status);
  assert.equal(response.data.code, 200, route + ": " + response.data.message);
  return response.data.result;
}
async function revoked(token) {
  const response = await request("GET", "/me", undefined, token);
  assert.equal(response.status, 401, "Revoked session must receive HTTP 401");
  assert.equal(response.data.code, 100003);
}
async function login(username, password) {
  for (let attempt = 1; attempt <= 3; attempt++) {
    const challenge = await ok("GET", "/randomImage", undefined, "");
    assert.ok(challenge.captchaId && challenge.captchaImg?.startsWith("data:image/"));
    const imageFile = path.join(folder, "captcha-" + (++sequence) + ".png");
    fs.writeFileSync(imageFile, Buffer.from(challenge.captchaImg.split(",")[1], "base64"));
    console.log("CAPTCHA_IMAGE=" + imageFile);
    const answer = (await input.question("Enter image CAPTCHA for " + username + " [" + attempt + "/3] (or skip): ")).trim();
    if (answer.toLowerCase() === "skip") throw new Skip("CAPTCHA login skipped by operator");
    const response = await request("POST", "/login", { username, password, captchaId: challenge.captchaId, captcha: answer }, "");
    // Retry only the explicit CAPTCHA error, never password/auth/service failures.
    if (response.status === 200 && response.data.code === 200007 && attempt < 3) {
      console.log("CAPTCHA was rejected; requesting a fresh image for another attempt.");
      continue;
    }
    assert.equal(response.status, 200, "/login HTTP " + response.status);
    assert.equal(response.data.code, 200, "/login: " + response.data.message);
    const session = response.data.result;
    assert.ok(session.accessToken && session.userInfo.userName === username);
    return session.accessToken;
  }
}
async function createUser(suffix) {
  const username = prefix + "_" + suffix;
  const password = "Session-test-123!";
  await ok("POST", "/register", { userName: username, passWord: password, nickName: username, authorityId: roles[0], authorityIds: [roles[0]], enable: 1 });
  const lookup = async () => (await ok("GET", "/getUserList?pageNo=1&pageSize=100&keyword=" + encodeURIComponent(username))).list.find((user) => user.userName === username);
  cleanups.push({ label: username, run: async () => { const user = await lookup(); if (user) await ok("DELETE", "/deleteUser", { userId: user.ID }); } });
  const user = await lookup();
  assert.ok(user, "Created user missing");
  return { ...user, username, password };
}
async function scenario(label, suffix, run) {
  if (selectedScenarios && !selectedScenarios.has(suffix)) return;
  try {
    const user = await createUser(suffix);
    const token = await login(user.username, user.password);
    await run(user, token);
    results.push({ label, status: "PASS" });
    console.log("PASS " + label);
  } catch (error) {
    const status = error instanceof Skip ? "SKIP" : "FAIL";
    results.push({ label, status, detail: error.message });
    console.log(status + " " + label + ": " + error.message);
  }
}
try {
  const admin = await ok("GET", "/me");
  assert.equal(admin.authorityId, 1, "Use the administrator recovery role for this regression");
  for (const role of roles) {
    await ok("POST", "/authority/createAuthority", { authorityId: role, authorityName: prefix + "_" + role, parentId: 0, defaultRouter: "index" });
    cleanups.push({ label: "Role " + role, run: () => ok("DELETE", "/authority/deleteAuthority", { id: role }) });
  }
  await scenario("Current user, unchanged roles and ordinary-role HTTP 403", "profile", async (user, token) => {
    const me = await ok("GET", "/me", undefined, token);
    assert.equal(me.ID, user.ID); assert.equal(me.password, undefined);
    assert.equal((await request("GET", "/getUserList", undefined, token)).status, 403);
    await ok("PUT", "/updateUserInfo", { ID: user.ID, nickName: "Updated display", authorityId: roles[0], authorityIds: [roles[0]], enable: 1 });
    assert.equal((await ok("GET", "/me", undefined, token)).nickName, "Updated display");
  });
  await scenario("Password validation, change and fresh login", "password", async (user, token) => {
    const wrong = await request("PUT", "/changePassword", { oldPassword: "wrong-password", newPassword: "Changed-test-456!" }, token);
    assert.equal(wrong.status, 200); assert.notEqual(wrong.data.code, 200); assert.notEqual(wrong.data.code, 100003);
    await ok("GET", "/me", undefined, token);
    await ok("PUT", "/changePassword", { oldPassword: user.password, newPassword: "Changed-test-456!" }, token);
    await revoked(token);
    const fresh = await login(user.username, "Changed-test-456!");
    await ok("GET", "/me", undefined, fresh);
  });
  await scenario("Freeze then unfreeze never restores the old token", "freeze", async (user, token) => {
    await ok("PUT", "/updateUserInfo", { ID: user.ID, enable: 2 }); await revoked(token);
    await ok("PUT", "/updateUserInfo", { ID: user.ID, enable: 1 }); await revoked(token);
  });
  await scenario("Default role change revokes existing token", "default_role", async (user, token) => {
    await ok("PUT", "/updateUserInfo", { ID: user.ID, authorityId: roles[1], authorityIds: roles }); await revoked(token);
  });
  await scenario("Associated role change revokes existing token", "linked_role", async (user, token) => {
    await ok("PUT", "/updateUserInfo", { ID: user.ID, authorityId: roles[0], authorityIds: roles }); await revoked(token);
  });
  await scenario("Password reset revokes and default reset password can log in", "reset", async (user, token) => {
    await ok("PUT", "/resetUserPassword", { userId: user.ID }); await revoked(token);
    const fresh = await login(user.username, process.env.GOZERO_RESET_PASSWORD || "goZero");
    await ok("GET", "/me", undefined, fresh);
  });
  await scenario("Logout replay cannot revoke a later login", "logout", async (user, token) => {
    await ok("POST", "/logout", {}, token); await revoked(token);
    const fresh = await login(user.username, user.password);
    const replay = await request("POST", "/logout", {}, token);
    assert.equal(replay.status, 401);
    await ok("GET", "/me", undefined, fresh);
  });
  await scenario("Deleting a user revokes existing token", "delete", async (user, token) => {
    await ok("DELETE", "/deleteUser", { userId: user.ID }); await revoked(token);
  });
} catch (error) {
  results.push({ label: "Setup", status: "FAIL", detail: error.message });
} finally {
  input.close();
  for (const item of cleanups.reverse()) {
    try { await item.run(); results.push({ label: "Cleanup " + item.label, status: "PASS" }); }
    catch (error) { results.push({ label: "Cleanup " + item.label, status: "FAIL", detail: error.message }); }
  }
  fs.mkdirSync("test/reports", { recursive: true });
  const reportFile = selectedScenarios
    ? "test/reports/" + prefix + ".json"
    : "test/reports/latest-session-smoke.json";
  fs.writeFileSync(reportFile, JSON.stringify({ time: new Date().toISOString(), base, prefix, scenarios: selectedScenarios ? [...selectedScenarios] : allScenarios, results }, null, 2));
  console.log("Report: " + reportFile);
  if (results.some((item) => item.status === "FAIL")) process.exitCode = 1;
  else if (results.some((item) => item.status === "SKIP")) process.exitCode = 2;
}
