import { test } from "node:test";
import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { SkillGildClient, SkillGildError } from "../dist/index.js";

function fake(status, body, headers = {}) {
  const calls = [];
  const fetch = async (url, init) => {
    calls.push({ url, init });
    return new Response(body === undefined ? null : JSON.stringify(body), { status, headers });
  };
  return { fetch, calls };
}

test("unwraps the data envelope and sends no key on public calls", async () => {
  const { fetch, calls } = fake(200, { data: { items: [{ slug: "a" }], next_cursor: "c2" } });
  const client = new SkillGildClient({ apiKey: "sk", fetch, baseUrl: "https://api.test/v1/" });
  const page = await client.listSkills({ query: " design ", cursor: "c1", limit: 10 });
  assert.equal(page.next_cursor, "c2");
  assert.equal(calls[0].url, "https://api.test/v1/skills?limit=10&q=design&cursor=c1");
  assert.equal(calls[0].init.headers.get("Authorization"), null);
});

test("sends the key and idempotency key on runs", async () => {
  const { fetch, calls } = fake(200, { data: { output: "ok" } });
  const client = new SkillGildClient({ apiKey: "sk", fetch });
  const run = await client.runSkill("a b", { prompt: "x" }, { idempotencyKey: "k1" });
  assert.equal(run.output, "ok");
  assert.equal(calls[0].url, "https://api.skillgild.dev/v1/skills/a%20b/run");
  assert.equal(calls[0].init.headers.get("Authorization"), "Bearer sk");
  assert.equal(calls[0].init.headers.get("Idempotency-Key"), "k1");
  assert.deepEqual(JSON.parse(calls[0].init.body), { input: { prompt: "x" } });
});

test("requires a key before calling signed-in endpoints", async () => {
  const { fetch, calls } = fake(200, {});
  await assert.rejects(new SkillGildClient({ fetch }).me(), /API key is required/);
  assert.equal(calls.length, 0);
});

test("maps error envelopes and Retry-After", async () => {
  const { fetch } = fake(429, { error: { code: "quota_exceeded", message: "Monthly runs used" } }, { "Retry-After": "120" });
  const error = await new SkillGildClient({ apiKey: "sk", fetch }).runSkill("a", {}).catch((e) => e);
  assert.ok(error instanceof SkillGildError);
  assert.equal(error.status, 429);
  assert.equal(error.code, "quota_exceeded");
  assert.equal(error.message, "Monthly runs used");
  assert.equal(error.retryAfter, 120);
});

test("handles 401 and non-JSON proxy errors", async () => {
  const unauthorized = fake(401, { error: { code: "unauthorized", message: "Invalid API key" } });
  const e1 = await new SkillGildClient({ apiKey: "bad", fetch: unauthorized.fetch }).me().catch((e) => e);
  assert.equal(e1.status, 401);
  assert.equal(e1.code, "unauthorized");
  const proxy = { fetch: async () => new Response("<html>bad gateway</html>", { status: 502 }) };
  const e2 = await new SkillGildClient({ fetch: proxy.fetch }).getSkill("a").catch((e) => e);
  assert.equal(e2.status, 502);
  assert.equal(e2.code, undefined);
});

test("returns undefined for 204 responses", async () => {
  const { fetch, calls } = fake(204);
  assert.equal(await new SkillGildClient({ apiKey: "sk", fetch }).endSession("s1"), undefined);
  assert.equal(calls[0].init.method, "DELETE");
});

test("covers catalog, device and key endpoints", async () => {
  const { fetch, calls } = fake(200, { data: [] });
  const client = new SkillGildClient({ apiKey: "sk", fetch });
  await client.listCategories();
  await client.listTags();
  await client.listCollections();
  await client.getCollection("starter kit");
  await client.featuredSkills();
  await client.startDeviceAuthorization("laptop");
  await client.pollDeviceAuthorization("dc");
  await client.revokeCurrentKey();
  assert.deepEqual(
    calls.map((c) => `${c.init.method ?? "GET"} ${new URL(c.url).pathname}${new URL(c.url).search}`),
    [
      "GET /v1/categories",
      "GET /v1/tags",
      "GET /v1/collections",
      "GET /v1/collections/starter%20kit",
      "GET /v1/featured-skills?slot=home",
      "POST /v1/device-authorizations",
      "POST /v1/device-authorizations/token",
      "DELETE /v1/me/api-keys/current",
    ],
  );
  assert.deepEqual(JSON.parse(calls[5].init.body), { device_name: "laptop", client_type: "skillgild-sdk" });
});

test("rejects base URLs with credentials or other schemes", () => {
  assert.throws(() => new SkillGildClient({ baseUrl: "https://u:p@api.test" }));
  assert.throws(() => new SkillGildClient({ baseUrl: "ftp://api.test" }));
});

test("times out a hung request", async () => {
  const fetch = (_url, init) => new Promise((_, reject) => init.signal.addEventListener("abort", () => reject(init.signal.reason)));
  await assert.rejects(new SkillGildClient({ fetch, timeoutMs: 20 }).getSkill("a"), { name: "TimeoutError" });
});

test("the CommonJS build loads", () => {
  const { SkillGildClient: Cjs } = createRequire(import.meta.url)("../dist/cjs/index.js");
  assert.equal(typeof Cjs, "function");
});
