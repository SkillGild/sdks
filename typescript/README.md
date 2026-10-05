<a href="https://skillgild.dev/?utm_source=npm&utm_medium=referral&utm_campaign=sdk"><img src="https://raw.githubusercontent.com/SkillGild/sdks/main/.github/assets/logo.png" alt="SkillGild" width="64" height="64" align="right"></a>

# @skillgild/sdk

Official TypeScript and JavaScript client for **[SkillGild](https://skillgild.dev/?utm_source=npm&utm_medium=referral&utm_campaign=sdk)**, the marketplace for hosted AI agent skills. Search the catalog, run hosted skills and drive hybrid tool sessions from Node.js 20+, Bun, Deno or the browser. Ships ESM and CommonJS builds with full type definitions and no runtime dependencies.

```sh
npm install @skillgild/sdk
```

Until the first npm release, build from source: `git clone https://github.com/SkillGild/sdks && cd sdks/typescript && npm install && npm run build`.

## Usage

```ts
import { SkillGildClient, SkillGildError } from "@skillgild/sdk";

const client = new SkillGildClient({ apiKey: process.env.SKILLGILD_API_KEY });

// Browse the catalog (no key needed)
const { items, next_cursor } = await client.listSkills({ query: "video", limit: 20 });
const skill = await client.getSkill("logo-design"); // includes media, tools and page copy

// Hybrid skills: your agent follows the guide and calls SkillGild server tools
const session = await client.startSession("logo-design");
const check = await client.callTool(session.session_id, "validate_svg", { svg: "<svg .../>" });
await client.endSession(session.session_id);

// Hosted skills run in one call; reuse the idempotency key only when retrying
try {
  const run = await client.runSkill("some-hosted-skill", { prompt: "..." }, { idempotencyKey: crypto.randomUUID() });
  console.log(run.output, run.usage);
} catch (error) {
  if (error instanceof SkillGildError && error.status === 429) console.log("retry in", error.retryAfter, "s");
  else throw error;
}
```

## API

| Method | Description |
| --- | --- |
| `listSkills({ query?, limit?, cursor? })` | One catalog page with `next_cursor` |
| `searchSkills(query?)` | First 100 matching skills |
| `getSkill(idOrSlug)` | Skill detail, tools, media and page copy |
| `listCategories()` · `listTags()` · `listCollections()` · `getCollection(slug)` · `featuredSkills(slot?)` | Marketplace content |
| `runSkill(idOrSlug, input, { idempotencyKey? })` | Run a hosted skill |
| `startSession(idOrSlug)` · `callTool(sessionId, tool, input)` · `endSession(sessionId)` | Hybrid tool sessions |
| `me()` · `revokeCurrentKey()` | The API key's account; revoke the key |
| `startDeviceAuthorization(name)` · `pollDeviceAuthorization(code)` | Device login that issues an API key |

Options: `apiKey`, `baseUrl` (default `https://api.skillgild.dev/v1`), `fetch` and `timeoutMs` (default 100 s). Every method takes an optional `AbortSignal`. Errors are `SkillGildError` with `status`, `code` and `retryAfter`.

Full reference: [SkillGild API documentation](https://skillgild.dev/docs/api?utm_source=npm&utm_medium=referral&utm_campaign=sdk) · [Skill catalog](https://skillgild.dev/skills?utm_source=npm&utm_medium=referral&utm_campaign=sdk) · [Source and other languages](https://github.com/SkillGild/sdks)

MIT © SkillGild, Inc.
