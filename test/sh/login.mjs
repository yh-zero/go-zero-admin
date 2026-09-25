// Local developer login helper. Does not skip or solve CAPTCHA.
// 1. node test/sh/login.mjs image [username]
// 2. Read the generated PNG, then: node test/sh/login.mjs login <captcha>
// Session data is written only under the OS temporary directory, never logged.
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
const url = process.env.GOZERO_TEST_URL || "http://127.0.0.1:7001";
if (!/^http:\/\/(127\.0\.0\.1|localhost):\d+$/.test(url))
  throw new Error("Only local development APIs are supported");
const folder = path.join(os.tmpdir(), "go-zero-admin-regression");
fs.mkdirSync(folder, { recursive: true });
const imageFile = path.join(folder, "captcha.png");
const challengeFile = path.join(folder, "challenge.json");
const sessionFile = path.join(folder, "session.json");
const action = process.argv[2];
if (action === "image") {
  const res = await fetch(url + "/v1/sys/randomImage");
  const data = await res.json();
  if (data.code !== 200) throw Error(data.message);
  const { captchaId, captchaImg } = data.result;
  if (!captchaId || !captchaImg.startsWith("data:image/")) throw Error("Invalid captcha contract");
  fs.writeFileSync(imageFile, Buffer.from(captchaImg.split(",")[1], "base64"));
  fs.writeFileSync(
    challengeFile,
    JSON.stringify({
      captchaId,
      username: process.argv[3] || process.env.GOZERO_TEST_USER || "admin",
    }),
  );
  console.log(imageFile);
} else if (action === "login") {
  const challenge = JSON.parse(fs.readFileSync(challengeFile, "utf8"));
  const password = process.env.GOZERO_TEST_PASSWORD || "123456";
  const res = await fetch(url + "/v1/sys/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      username: challenge.username,
      password,
      captchaId: challenge.captchaId,
      captcha: process.argv[3],
    }),
  });
  const data = await res.json();
  if (data.code !== 200) throw Error(data.message);
  fs.writeFileSync(
    sessionFile,
    JSON.stringify({
      accessToken: data.result.accessToken,
      accessExpire: data.result.accessExpire,
    }),
  );
  console.log("Session saved: " + sessionFile);
} else throw Error("Usage: login.mjs image [username] | login <captcha>");
