// Run against a local developer API only. Creates isolated smoke_* records and removes them.
// Set GOZERO_TEST_TOKEN or pass a JSON session file containing accessToken.
// Optional TEST_MAIL_TO explicitly enables a real test email; otherwise email success is skipped.
import fs from "node:fs";
import assert from "node:assert/strict";

const base = process.env.GOZERO_TEST_URL || "http://127.0.0.1:7001";
if (!/^http:\/\/(127\.0\.0\.1|localhost):\d+$/.test(base))
  throw new Error("Only local development APIs are supported");
const sessionFile = process.argv[2];
const token =
  process.env.GOZERO_TEST_TOKEN ||
  (sessionFile && JSON.parse(fs.readFileSync(sessionFile, "utf8")).accessToken);
if (!token) throw new Error("Provide an authenticated development session");
const prefix = "smoke_" + Date.now();
const report = [];
const requestedOperations = new Set();
const cleanup = [];
const roleId = Math.floor(Date.now() / 1000) + 100000;
let menuId, userId, dictionaryId, itemId, apiId, bulkId;
const record = (label, status, detail = "") => {
  report.push({ label, status, detail });
  console.log(status + " " + label + (detail ? ": " + detail : ""));
};
async function call(method, path, body, expectFailure = false) {
  requestedOperations.add(method + " /v1/sys" + path.split("?")[0]);
  const res = await fetch(base + "/v1/sys" + path, {
    method,
    headers: { "Content-Type": "application/json", Authorization: "Bearer " + token },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(35000),
  });
  const data = await res.json().catch(() => ({}));
  if (res.status === 200)
    assert.ok(Number.isInteger(data.code), "Missing response envelope: " + path);
  if (expectFailure) {
    assert.ok(
      ![401, 403].includes(res.status) && data.code !== 100003,
      "Validation request was blocked by authentication: " + path,
    );
    assert.ok(res.status >= 400 || data.code !== 200, "Expected failure: " + path);
    return data;
  }
  assert.equal(res.status, 200, method + " " + path + " HTTP " + res.status);
  assert.equal(data.code, 200, method + " " + path + ": " + (data.message || "invalid response"));
  return data.result;
}
async function step(label, fn) {
  try {
    await fn();
    record(label, "PASS");
  } catch (error) {
    record(label, "FAIL", error.message);
  }
}
function flatten(nodes = []) {
  return (nodes || []).flatMap((row) => [row, ...flatten(row.children)]);
}
function findId(rows, predicate) {
  const row = rows.find(predicate);
  assert.ok(row, "Created record missing");
  return row.ID;
}
function removeLater(label, method, path, body) {
  cleanup.push({ label, method, path, body });
}
function forgetCleanup(label) {
  const index = cleanup.findIndex((x) => x.label === label);
  if (index >= 0) cleanup.splice(index, 1);
}
const menuBody = {
  parentId: 0,
  path: prefix,
  name: prefix,
  hidden: true,
  component: "views/index.vue",
  sort: 0,
  parameters: [],
  menuBtn: [{ name: "edit", desc: "Smoke edit" }],
  meta: { title: prefix, icon: "lucide:flask-conical", keepAlive: false, closeTab: false },
};
try {
  await step("Current menu /menu/getMenu", async () => {
    const r = await call("GET", "/menu/getMenu");
    assert.ok("menus" in r);
  });
  await step("Role create/list/update and new-role GET policy", async () => {
    await call("POST", "/authority/createAuthority", {
      authorityId: roleId,
      authorityName: prefix,
      parentId: 0,
      defaultRouter: "index",
    });
    removeLater("Role", "DELETE", "/authority/deleteAuthority", { id: roleId });
    const roles = await call("GET", "/authority/getAuthorityList?pageNo=1&pageSize=500");
    assert.ok(flatten(roles.list).some((x) => x.authorityId === roleId));
    await call("PUT", "/authority/updateAuthority", {
      authorityId: roleId,
      authorityName: prefix + "_edited",
      parentId: 0,
      defaultRouter: "index",
    });
    const policies = await call("GET", "/casbin/getPathByAuthorityId?authorityId=" + roleId);
    assert.ok(policies.list.some((x) => x.path === "/v1/sys/menu/getMenu" && x.method === "GET"));
  });
  await step("Menu create/tree/list/detail/update/cycle validation", async () => {
    await call("POST", "/menu/addBaseMenu", menuBody);
    const r = await call("GET", "/menu/getMenuList");
    menuId = findId(flatten(r.list), (x) => x.name === prefix);
    removeLater("Menu", "DELETE", "/menu/deleteBaseMenu", { id: menuId });
    const detail = await call("GET", "/menu/getBaseMenuById?id=" + menuId);
    assert.equal(detail.sysBaseMenu.name, prefix);
    await call("GET", "/menu/getBaseMenuTree");
    await call("PUT", "/menu/updateBaseMenu", {
      ...menuBody,
      ID: menuId,
      meta: { ...menuBody.meta, title: prefix + "_edited" },
    });
    const updated = (await call("GET", "/menu/getBaseMenuById?id=" + menuId)).sysBaseMenu;
    assert.equal(updated.meta.title, prefix + "_edited");
    await call("PUT", "/menu/updateBaseMenu", { ...menuBody, ID: menuId, parentId: menuId }, true);
  });
  await step("Role menu grant/target-role read/buttons/clear", async () => {
    assert.ok(menuId);
    await call("POST", "/authority/addAuthorityMenu", {
      authorityId: roleId,
      menuIds: String(menuId),
    });
    const selected = await call("GET", "/menu/getMenuAuthority?authorityId=" + roleId);
    assert.ok(flatten(selected.list).some((x) => x.ID === menuId || Number(x.menuId) === menuId));
    const detail = (await call("GET", "/menu/getBaseMenuById?id=" + menuId)).sysBaseMenu;
    const btnId = detail.menuBtn[0].ID;
    await call("PUT", "/menu/updateAuthorityButtons", { authorityId: roleId, menuBtnIds: [btnId] });
    assert.deepEqual(
      (await call("GET", "/menu/getAuthorityButtons?authorityId=" + roleId)).menuBtnIds,
      [btnId],
    );
    await call("PUT", "/menu/updateAuthorityButtons", { authorityId: roleId, menuBtnIds: [] });
    await call("POST", "/authority/addAuthorityMenu", { authorityId: roleId, menuIds: "" });
    assert.equal(
      flatten((await call("GET", "/menu/getMenuAuthority?authorityId=" + roleId)).list).length,
      0,
    );
  });
  await step("User register/list/search/edit/clear/reset/freeze", async () => {
    await call("POST", "/register", {
      userName: prefix,
      passWord: "Smoke-test-123!",
      nickName: prefix,
      authorityId: roleId,
      authorityIds: [roleId],
      enable: 1,
      phone: "12345678901",
      email: "test@example.test",
    });
    userId = findId(
      (await call("GET", "/getUserList?pageNo=1&pageSize=100&keyword=" + prefix)).list,
      (x) => x.userName === prefix,
    );
    removeLater("User", "DELETE", "/deleteUser", { userId });
    await call(
      "POST",
      "/register",
      {
        userName: prefix,
        passWord: "Smoke-test-123!",
        nickName: prefix,
        authorityId: roleId,
        authorityIds: [roleId],
        enable: 1,
      },
      true,
    );
    const found = await call("GET", "/getUserList?pageNo=1&pageSize=100&keyword=" + prefix);
    assert.ok(found.list.every((x) => x.userName.includes(prefix) || x.nickName.includes(prefix)));
    await call("PUT", "/updateUserInfo", {
      ID: userId,
      nickName: prefix + "_edited",
      phone: "",
      email: "",
      authorityId: roleId,
      authorityIds: [roleId],
      enable: 2,
    });
    const user = (
      await call("GET", "/getUserList?pageNo=1&pageSize=100&keyword=" + prefix)
    ).list.find((x) => x.ID === userId);
    assert.equal(user.phone, "");
    assert.equal(user.email, "");
    assert.equal(user.enable, 2);
    assert.equal(user.nickName, prefix + "_edited");
    await call("PUT", "/updateUserInfo", { ID: userId, authorityIds: [] }, true);
    await call("PUT", "/resetUserPassword", { userId });
    await call("PUT", "/updateUserInfo", { ID: userId, enable: 1 });
  });
  await step("API create/list/all/update/sort error", async () => {
    const data = {
      path: "/v1/sys/" + prefix,
      method: "GET",
      description: prefix,
      apiGroup: prefix,
    };
    await call("POST", "/api/createApi", data);
    apiId = findId((await call("GET", "/api/getAllApiList")).apiList, (x) => x.path === data.path);
    removeLater("API", "DELETE", "/api/deleteApi", { ID: apiId });
    await call("GET", "/api/getApiList?pageNo=1&pageSize=10&apiGroup=" + prefix);
    await call("PUT", "/api/updateApi", { ...data, ID: apiId, description: prefix + "_edited" });
    await call(
      "GET",
      "/api/getApiList?pageNo=1&pageSize=10&orderKey=invalid_order",
      undefined,
      true,
    );
    await call("POST", "/api/createApi", data, true);
  });
  await step("Casbin path save / ID save / invalid ID rollback / clear", async () => {
    await call("PUT", "/casbin/updateCasbinData", {
      authorityId: roleId,
      casbinInfoList: [{ path: "/v1/sys/menu/getMenu", method: "GET" }],
    });
    let r = await call("GET", "/casbin/getPathByAuthorityId?authorityId=" + roleId);
    assert.ok(r.list.some((x) => x.path === "/v1/sys/menu/getMenu"));
    assert.ok(apiId);
    await call("PUT", "/casbin/updateCasbinDataByApiIds", { authorityId: roleId, apiIds: [apiId] });
    r = await call("GET", "/casbin/getPathByAuthorityId?authorityId=" + roleId);
    assert.ok(r.list.some((x) => x.path === "/v1/sys/" + prefix));
    await call(
      "PUT",
      "/casbin/updateCasbinDataByApiIds",
      { authorityId: roleId, apiIds: [apiId, 9007199254740990] },
      true,
    );
    r = await call("GET", "/casbin/getPathByAuthorityId?authorityId=" + roleId);
    assert.ok(
      r.list.some((x) => x.path === "/v1/sys/" + prefix),
      "invalid request must preserve policies",
    );
    await call("PUT", "/casbin/updateCasbinDataByApiIds", { authorityId: roleId, apiIds: [] });
    assert.equal(
      ((await call("GET", "/casbin/getPathByAuthorityId?authorityId=" + roleId)).list || []).length,
      0,
    );
  });
  await step("API batch delete", async () => {
    const data = {
      path: "/v1/sys/" + prefix + "_bulk",
      method: "GET",
      description: prefix,
      apiGroup: prefix,
    };
    await call("POST", "/api/createApi", data);
    bulkId = findId((await call("GET", "/api/getAllApiList")).apiList, (x) => x.path === data.path);
    removeLater("Bulk API fallback", "DELETE", "/api/deleteApi", { ID: bulkId });
    await call("DELETE", "/api/deleteApisByIds", { ids: [bulkId] });
    forgetCleanup("Bulk API fallback");
    assert.ok(!(await call("GET", "/api/getAllApiList")).apiList.some((x) => x.ID === bulkId));
  });
  await step("Dictionary create/list/details/update", async () => {
    await call("POST", "/dictionary/createSysDictionary", {
      name: prefix,
      type: prefix,
      status: 1,
      desc: prefix,
    });
    dictionaryId = findId(
      (await call("GET", "/dictionary/getSysDictionaryList")).list,
      (x) => x.type === prefix,
    );
    removeLater("Dictionary", "DELETE", "/dictionary/deleteSysDictionary", { id: dictionaryId });
    const detail = await call(
      "GET",
      "/dictionary/getSysDictionaryDetails?id=" + dictionaryId + "&status=1",
    );
    assert.equal(detail.type, prefix);
    await call("PUT", "/dictionary/updateSysDictionary", {
      ID: dictionaryId,
      name: prefix + "_edited",
      type: prefix,
      status: 1,
      desc: "",
    });
    assert.equal(
      (await call("GET", "/dictionary/getSysDictionaryDetails?type=" + prefix + "&status=1")).desc,
      "",
    );
  });
  await step("Dictionary item create/list/detail/zero-value update", async () => {
    assert.ok(dictionaryId);
    await call("POST", "/dictionary/createSysDictionaryInfo", {
      label: prefix,
      value: 1,
      extend: "x",
      status: 1,
      sort: 5,
      sysDictionaryID: dictionaryId,
    });
    itemId = findId(
      (
        await call(
          "GET",
          "/dictionary/getSysDictionaryInfoList?sysDictionaryID=" +
            dictionaryId +
            "&pageNo=1&pageSize=10",
        )
      ).list,
      (x) => x.label === prefix,
    );
    removeLater("Dictionary item", "DELETE", "/dictionary/deleteSysDictionaryInfo", { id: itemId });
    await call("GET", "/dictionary/getSysDictionaryInfoListDetailsById?id=" + itemId);
    await call("PUT", "/dictionary/updateSysDictionaryInfo", {
      ID: itemId,
      label: prefix,
      value: 0,
      extend: "",
      status: 1,
      sort: 0,
      sysDictionaryID: dictionaryId,
    });
    const detail = await call(
      "GET",
      "/dictionary/getSysDictionaryInfoListDetailsById?id=" + itemId,
    );
    assert.equal(detail.value, 0);
    assert.equal(detail.sort, 0);
    assert.equal(detail.extend, "");
    const zero = await call(
      "GET",
      "/dictionary/getSysDictionaryInfoList?sysDictionaryID=" +
        dictionaryId +
        "&value=0&pageNo=1&pageSize=10",
    );
    assert.ok(zero.list.some((x) => x.ID === itemId));
  });
  await step("Email invalid recipient fails clearly", async () => {
    await call("POST", "/base/sendEmailCode", { email: "invalid", isForce: false }, true);
  });
  if (process.env.TEST_MAIL_TO)
    await step("Email real send", () =>
      call("POST", "/base/sendEmailCode", { email: process.env.TEST_MAIL_TO, isForce: false }),
    );
  else record("Email real delivery", "SKIP", "No configured recipient / SMTP supplied");
  await step("Upload invalid file rejected", async () => {
    requestedOperations.add("POST /v1/sys/base/uploadFileImg");
    const form = new FormData();
    form.append(
      "file_img",
      new Blob(["<html>not an image</html>"], { type: "image/png" }),
      "fake.png",
    );
    const res = await fetch(base + "/v1/sys/base/uploadFileImg", {
      method: "POST",
      headers: { Authorization: "Bearer " + token },
      body: form,
    });
    const data = await res.json();
    assert.equal(res.status, 200);
    assert.notEqual(data.code, 200);
    assert.notEqual(data.code, 100003);
  });
  record(
    "Upload real OSS success",
    "SKIP",
    "Requires configured OSS; verify separately if available",
  );
  await step("Invalid JWT rejected", async () => {
    const res = await fetch(base + "/v1/sys/menu/getMenu", {
      headers: { Authorization: "Bearer invalid.token" },
    });
    const data = await res.json();
    assert.ok(res.status === 401 || data.code === 100003);
  });
} finally {
  // Reverse creation order first; users and role-menu links must be removed before menus/roles.
  if (userId) {
    try {
      await call("DELETE", "/deleteUser", { userId });
      forgetCleanup("User");
      record("User delete", "PASS");
    } catch (e) {
      record("User delete", "FAIL", e.message);
    }
  }
  if (menuId) {
    try {
      await call("PUT", "/menu/updateAuthorityButtons", { authorityId: roleId, menuBtnIds: [] });
      await call("POST", "/authority/addAuthorityMenu", { authorityId: roleId, menuIds: "" });
    } catch {}
  }
  for (const item of [...cleanup].reverse()) {
    try {
      await call(item.method, item.path, item.body);
      record(item.label + " delete", "PASS");
    } catch (e) {
      record(item.label + " cleanup", "FAIL", e.message);
    }
  }
  const output =
    process.env.GOZERO_TEST_REPORT || new URL("../reports/latest-smoke.json", import.meta.url);
  fs.mkdirSync(new URL("../reports/", import.meta.url), { recursive: true });
  fs.writeFileSync(
    output,
    JSON.stringify(
      {
        prefix,
        finishedAt: new Date().toISOString(),
        requestedOperations: [...requestedOperations].sort(),
        report,
      },
      null,
      2,
    ),
  );
}
if (report.some((x) => x.status === "FAIL")) process.exitCode = 1;
