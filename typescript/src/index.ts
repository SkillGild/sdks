/** Card or gallery media for a skill. */
export type Media = {
  kind: "image" | "video" | string;
  url: string;
  poster_url?: string;
  preview_url?: string;
  alt: string;
  credit?: string;
  width?: number;
  height?: number;
};

export type FAQ = { q: string; a: string };

/** Long-form skill page copy (getSkill only). */
export type SkillPageContent = {
  what_you_get?: string[];
  good_fit?: string[];
  not_for?: string[];
  requirements?: string[];
  faqs?: FAQ[];
};

export type Skill = {
  id: string;
  slug: string;
  name: string;
  description: string;
  distribution_mode: string;
  access_tier: "free" | "paid";
  price_amount_minor: number;
  price_currency: string;
  /** True for a paid skill that a SkillGild Pro subscription includes. */
  included_in_pro: boolean;
  free_runs_per_month: number;
  current_version: string;
  input_schema: Record<string, unknown>;
  /** "prompt_pipeline" runs on SkillGild; "hybrid_tools" runs in your agent with server tools. */
  runtime_type: "prompt_pipeline" | "hybrid_tools";
  /** Public interface of a hybrid skill's server tools (getSkill only). */
  tools?: SkillTool[];
  visibility?: "public" | "unlisted";
  updated_at?: string;
  category?: string;
  category_slug?: string;
  creator_name?: string;
  /** Provenance for skills packaged from an upstream repository. */
  source_url?: string;
  license?: string;
  attribution?: string;
  /** Card media (catalog listings only). */
  cover?: Media;
  /** Full ordered media gallery (getSkill only). */
  media?: Media[];
  /** Long-form page copy (getSkill only). */
  page?: SkillPageContent;
};

/** The shorter skill shape used by featured skills and collections. */
export type SkillSummary = {
  id: string;
  slug: string;
  name: string;
  description: string;
  category: string;
  creator_name: string;
  visibility: string;
  access_tier: "free" | "paid";
  price_amount_minor: number;
  price_currency: string;
  free_runs_per_month: number;
  updated_at: string;
  cover?: Media;
};

/** One page of the catalog. Pass `next_cursor` to `listSkills` for the following page. */
export type SkillPage = {
  items: Skill[];
  next_cursor?: string;
};

export type ListSkillsOptions = {
  query?: string;
  /** 1 to 100; the API default is 50. */
  limit?: number;
  cursor?: string;
  signal?: AbortSignal;
};

export type Category = {
  id: string;
  parent_id: string | null;
  slug: string;
  name: string;
  short_description: string;
  sort_order: number;
};

export type Tag = {
  id: string;
  slug: string;
  name: string;
  color: string;
  skill_count: number;
};

export type Collection = {
  id: string;
  slug: string;
  name: string;
  description: string;
  image_url: string;
  skill_count: number;
  updated_at: string;
};

export type CollectionDetail = {
  id: string;
  slug: string;
  name: string;
  description: string;
  image_url: string;
  skills: SkillSummary[];
};

export type SkillTool = {
  name: string;
  title: string;
  description: string;
  input_schema: Record<string, unknown>;
  example_input?: Record<string, unknown>;
};

export type Usage = {
  used: number;
  limit: number | null;
  reset_at: string;
  access_tier: string;
};

export type AgentSession = {
  session_id: string;
  skill_id: string;
  skill_slug: string;
  skill_name: string;
  version: string;
  expires_at: string;
  max_tool_calls: number;
  tool_calls_used: number;
  /** True when an open session was returned instead of starting a new one. */
  resumed: boolean;
  /** Instructions for the agent to follow during this session. */
  guide: string;
  tools: SkillTool[];
  /** Present when this call started the session and so used one run. */
  usage?: Usage;
};

export type ToolCallResult<T = unknown> = {
  session_id: string;
  tool: string;
  result: T;
  tool_calls_used: number;
  tool_calls_remaining: number;
  expires_at: string;
};

export type SkillRun = {
  execution_id: string;
  skill_id: string;
  skill_slug: string;
  version: string;
  output: string;
  usage: Usage;
};

/** The account an API key belongs to. */
export type Account = {
  id: string;
  email: string;
  name: string;
  auth_type: "api_key" | "user_jwt";
};

/** A pending device login. Show `user_code` and `verification_uri` to the user. */
export type DeviceAuthorization = {
  device_code: string;
  user_code: string;
  verification_uri: string;
  expires_at: string;
  interval_seconds: number;
};

export type DeviceToken = {
  status: "authorization_pending" | "authorized";
  /** The issued API key, once authorized. */
  access_token?: string;
};

export type SkillGildOptions = {
  apiKey?: string;
  baseUrl?: string;
  fetch?: typeof globalThis.fetch;
  /** Per-request timeout in milliseconds. Defaults to 100 000; 0 disables it. */
  timeoutMs?: number;
};

type ErrorEnvelope = { error?: { code?: string; message?: string } };

export class SkillGildError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code?: string,
    /** Seconds to wait before retrying, from the Retry-After header (429 and 503). */
    readonly retryAfter?: number,
  ) {
    super(message);
    this.name = "SkillGildError";
  }
}

export const DEFAULT_BASE_URL = "https://api.skillgild.dev/v1";

export class SkillGildClient {
  private readonly baseUrl: string;
  private readonly apiKey?: string;
  private readonly fetcher: typeof globalThis.fetch;
  private readonly timeoutMs: number;

  constructor(options: SkillGildOptions = {}) {
    this.baseUrl = (options.baseUrl ?? DEFAULT_BASE_URL).replace(/\/+$/, "");
    const parsed = new URL(this.baseUrl);
    if (!(["https:", "http:"].includes(parsed.protocol)) || parsed.username || parsed.password) {
      throw new Error("baseUrl must be an absolute HTTP(S) URL without user information");
    }
    this.apiKey = options.apiKey;
    this.fetcher = options.fetch ?? globalThis.fetch;
    this.timeoutMs = options.timeoutMs ?? 100_000;
  }

  /** One page of the public catalog, with a cursor for the next page. */
  listSkills(options: ListSkillsOptions = {}): Promise<SkillPage> {
    const params = new URLSearchParams({ limit: String(options.limit ?? 100) });
    if (options.query?.trim()) params.set("q", options.query.trim());
    if (options.cursor) params.set("cursor", options.cursor);
    return this.request(`/skills?${params}`, { signal: options.signal });
  }

  /** The first 100 matching skills. Use `listSkills` to page through a larger catalog. */
  async searchSkills(query?: string, signal?: AbortSignal): Promise<Skill[]> {
    const page = await this.listSkills({ query, limit: 100, signal });
    return page.items;
  }

  getSkill(idOrSlug: string, signal?: AbortSignal): Promise<Skill> {
    return this.request(`/skills/${encodeURIComponent(idOrSlug)}`, { signal });
  }

  listCategories(signal?: AbortSignal): Promise<Category[]> {
    return this.request("/categories", { signal });
  }

  listTags(signal?: AbortSignal): Promise<Tag[]> {
    return this.request("/tags", { signal });
  }

  listCollections(signal?: AbortSignal): Promise<Collection[]> {
    return this.request("/collections", { signal });
  }

  getCollection(slug: string, signal?: AbortSignal): Promise<CollectionDetail> {
    return this.request(`/collections/${encodeURIComponent(slug)}`, { signal });
  }

  /** Skills featured in a placement slot ("home" by default). */
  featuredSkills(slot = "home", signal?: AbortSignal): Promise<SkillSummary[]> {
    return this.request(`/featured-skills?${new URLSearchParams({ slot })}`, { signal });
  }

  /** The account the configured API key belongs to. */
  me(signal?: AbortSignal): Promise<Account> {
    return this.request("/me", { signal, authenticated: true });
  }

  /** Revokes the API key this client authenticates with. Later calls with it fail with 401. */
  async revokeCurrentKey(signal?: AbortSignal): Promise<void> {
    await this.request("/me/api-keys/current", { method: "DELETE", signal, authenticated: true });
  }

  /**
   * Starts a device login. Show the user `user_code` and `verification_uri`, then call
   * `pollDeviceAuthorization` every `interval_seconds` until it returns an access token.
   */
  startDeviceAuthorization(
    deviceName: string,
    options: { clientType?: string; signal?: AbortSignal } = {},
  ): Promise<DeviceAuthorization> {
    return this.request("/device-authorizations", {
      method: "POST",
      body: JSON.stringify({ device_name: deviceName, client_type: options.clientType ?? "skillgild-sdk" }),
      signal: options.signal,
    });
  }

  pollDeviceAuthorization(deviceCode: string, signal?: AbortSignal): Promise<DeviceToken> {
    return this.request("/device-authorizations/token", {
      method: "POST",
      body: JSON.stringify({ device_code: deviceCode }),
      signal,
    });
  }

  runSkill(
    idOrSlug: string,
    input: Record<string, unknown>,
    options: { signal?: AbortSignal; idempotencyKey?: string } = {},
  ): Promise<SkillRun> {
    return this.request(`/skills/${encodeURIComponent(idOrSlug)}/run`, {
      method: "POST",
      body: JSON.stringify({ input }),
      signal: options.signal,
      authenticated: true,
      idempotencyKey: options.idempotencyKey,
    });
  }

  /** Starts (or resumes) a session for a hybrid_tools skill. A new session uses one run. */
  startSession(idOrSlug: string, signal?: AbortSignal): Promise<AgentSession> {
    return this.request(`/skills/${encodeURIComponent(idOrSlug)}/sessions`, {
      method: "POST",
      body: "{}",
      signal,
      authenticated: true,
    });
  }

  /** Calls a server tool in an open session. Each call uses one of the session's calls. */
  callTool<T = unknown>(
    sessionId: string,
    tool: string,
    input: Record<string, unknown>,
    signal?: AbortSignal,
  ): Promise<ToolCallResult<T>> {
    return this.request(
      `/skill-sessions/${encodeURIComponent(sessionId)}/tools/${encodeURIComponent(tool)}`,
      { method: "POST", body: JSON.stringify({ input }), signal, authenticated: true },
    );
  }

  async endSession(sessionId: string, signal?: AbortSignal): Promise<void> {
    await this.request(`/skill-sessions/${encodeURIComponent(sessionId)}`, {
      method: "DELETE",
      signal,
      authenticated: true,
    });
  }

  private async request<T>(
    path: string,
    options: RequestInit & { authenticated?: boolean; idempotencyKey?: string } = {},
  ): Promise<T> {
    const { authenticated, idempotencyKey, signal, ...init } = options;
    const headers = new Headers(init.headers);
    headers.set("Accept", "application/json");
    if (init.body) headers.set("Content-Type", "application/json");
    if (authenticated) {
      if (!this.apiKey) throw new Error("An API key is required for this SkillGild request.");
      headers.set("Authorization", `Bearer ${this.apiKey}`);
    }
    if (idempotencyKey) headers.set("Idempotency-Key", idempotencyKey);
    const signals = [signal, this.timeoutMs > 0 ? AbortSignal.timeout(this.timeoutMs) : undefined].filter(
      (s): s is AbortSignal => s !== undefined && s !== null,
    );
    const response = await this.fetcher(`${this.baseUrl}${path}`, {
      ...init,
      headers,
      signal: signals.length > 1 ? AbortSignal.any(signals) : signals[0],
    });
    if (response.status === 204) return undefined as T;
    const envelope = (await response.json().catch(() => ({}))) as { data?: T } & ErrorEnvelope;
    if (!response.ok) {
      const retryAfter = Number.parseInt(response.headers.get("Retry-After") ?? "", 10);
      throw new SkillGildError(
        envelope.error?.message ?? `SkillGild API request failed (${response.status})`,
        response.status,
        envelope.error?.code,
        Number.isFinite(retryAfter) ? retryAfter : undefined,
      );
    }
    return envelope.data as T;
  }
}
