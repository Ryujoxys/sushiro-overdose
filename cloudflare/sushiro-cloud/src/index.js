const JSON_HEADERS = {
  "content-type": "application/json; charset=utf-8",
  "cache-control": "no-store",
};

const REQUIRED_SECRETS = [
  "GITHUB_CLIENT_ID",
  "GITHUB_CLIENT_SECRET",
  "SESSION_SECRET",
  "TURSO_DATABASE_URL",
  "TURSO_AUTH_TOKEN",
];

const DATE_TYPES = ["weekday", "workday", "weekend", "holiday"];
// collector 的 rollup time_bucket 是 30 分钟粒度（HH:00 / HH:30，见
// collector/datetype.py）。此值会被桌面端 mergeQueueBaselineExports 用来覆盖
// 默认桶宽，报错会导致基准曲线桶口径与实际数据错位，必须与 collector 一致。
const DEFAULT_BUCKET_MINUTES = 30;

export default {
  async fetch(request, env) {
    try {
      assertRequiredEnv(env);
      const url = new URL(request.url);
      if (request.method === "OPTIONS") return new Response(null, { status: 204 });
      if (request.method === "GET" && url.pathname === "/health") {
        return json({ ok: true, service: "sushiro-cloud" });
      }
      if (request.method === "GET" && url.pathname === "/auth/github/start") {
        return await handleGitHubStart(request, env);
      }
      if (request.method === "GET" && url.pathname === "/auth/github/callback") {
        return await handleGitHubCallback(request, env);
      }
      if (request.method === "GET" && url.pathname === "/api/me") {
        const session = await requireSession(request, env);
        return json({
          ok: true,
          user: {
            id: Number(session.sub || 0),
            login: session.login || "",
            name: session.name || "",
            avatar_url: session.avatar_url || "",
          },
          session: { expires_at: new Date(session.exp * 1000).toISOString() },
        });
      }
      if (request.method === "GET" && url.pathname === "/api/queue/baseline/export") {
        await requireSession(request, env);
        return json(await buildQueueBaselineExport(env, null));
      }
      if (request.method === "GET" && url.pathname === "/api/queue/baseline/store") {
        await requireSession(request, env);
        const storeID = parsePositiveInt(url.searchParams.get("store_id") || url.searchParams.get("store"));
        if (!storeID) return json({ ok: false, error: "missing store_id" }, 400);
        return json(await buildQueueBaselineExport(env, storeID));
      }
      return json({ ok: false, error: "not found" }, 404);
    } catch (err) {
      const status = err && err.status ? err.status : 500;
      return json({ ok: false, error: String((err && err.message) || err) }, status);
    }
  },
};

async function handleGitHubStart(request, env) {
  const url = new URL(request.url);
  const returnTo = url.searchParams.get("return_to") || "";
  const clientState = url.searchParams.get("state") || "";
  if (!isAllowedLoopbackReturn(returnTo)) {
    throw httpError(400, "return_to must be a loopback URL");
  }
  if (!clientState) {
    throw httpError(400, "missing state");
  }
  const state = await signToken(
    {
      kind: "github_oauth_state",
      return_to: returnTo,
      client_state: clientState,
    },
    env.SESSION_SECRET,
    10 * 60,
  );
  const authURL = new URL("https://github.com/login/oauth/authorize");
  authURL.searchParams.set("client_id", env.GITHUB_CLIENT_ID);
  authURL.searchParams.set("redirect_uri", workerCallbackURL(request));
  authURL.searchParams.set("scope", "read:user");
  authURL.searchParams.set("state", state);
  return Response.redirect(authURL.toString(), 302);
}

async function handleGitHubCallback(request, env) {
  const url = new URL(request.url);
  const stateToken = url.searchParams.get("state") || "";
  let state;
  try {
    state = await verifyToken(stateToken, env.SESSION_SECRET);
  } catch (err) {
    return json({ ok: false, error: "invalid oauth state" }, 400);
  }
  if (state.kind !== "github_oauth_state" || !isAllowedLoopbackReturn(state.return_to)) {
    return json({ ok: false, error: "invalid oauth return" }, 400);
  }
  const returnURL = new URL(state.return_to);
  const fail = (message) => {
    returnURL.searchParams.set("state", state.client_state || "");
    returnURL.searchParams.set("error", message);
    return Response.redirect(returnURL.toString(), 302);
  };
  const code = url.searchParams.get("code") || "";
  if (!code) return fail(url.searchParams.get("error_description") || url.searchParams.get("error") || "missing code");

  const token = await exchangeGitHubCode(request, env, code);
  const user = await fetchGitHubUser(token.access_token);
  if (!isAllowedGitHubUser(user.login, env.ALLOWED_GITHUB_LOGINS || "")) {
    return fail("GitHub account is not allowed");
  }
  const ttl = parsePositiveInt(env.SESSION_TTL_SECONDS) || 30 * 24 * 60 * 60;
  const session = await signToken(
    {
      kind: "sushiro_cloud_session",
      sub: String(user.id || ""),
      login: user.login || "",
      name: user.name || "",
      avatar_url: user.avatar_url || "",
    },
    env.SESSION_SECRET,
    ttl,
  );
  const expiresAt = new Date((nowSeconds() + ttl) * 1000).toISOString();
  returnURL.searchParams.set("state", state.client_state || "");
  returnURL.searchParams.set("token", session);
  returnURL.searchParams.set("login", user.login || "");
  returnURL.searchParams.set("name", user.name || "");
  returnURL.searchParams.set("avatar_url", user.avatar_url || "");
  returnURL.searchParams.set("expires_at", expiresAt);
  return Response.redirect(returnURL.toString(), 302);
}

async function exchangeGitHubCode(request, env, code) {
  const body = new URLSearchParams();
  body.set("client_id", env.GITHUB_CLIENT_ID);
  body.set("client_secret", env.GITHUB_CLIENT_SECRET);
  body.set("code", code);
  body.set("redirect_uri", workerCallbackURL(request));
  const resp = await fetch("https://github.com/login/oauth/access_token", {
    method: "POST",
    headers: {
      accept: "application/json",
      "content-type": "application/x-www-form-urlencoded",
      "user-agent": "sushiro-cloud-worker",
    },
    body,
  });
  const data = await resp.json();
  if (!resp.ok || data.error || !data.access_token) {
    throw httpError(502, data.error_description || data.error || "GitHub token exchange failed");
  }
  return data;
}

async function fetchGitHubUser(accessToken) {
  const resp = await fetch("https://api.github.com/user", {
    headers: {
      accept: "application/vnd.github+json",
      authorization: `Bearer ${accessToken}`,
      "user-agent": "sushiro-cloud-worker",
      "x-github-api-version": "2022-11-28",
    },
  });
  const data = await resp.json();
  if (!resp.ok || !data.login) {
    throw httpError(502, data.message || "GitHub user fetch failed");
  }
  return data;
}

async function requireSession(request, env) {
  const header = request.headers.get("authorization") || "";
  const match = header.match(/^Bearer\s+(.+)$/i);
  if (!match) throw httpError(401, "missing bearer token");
  const payload = await verifyToken(match[1], env.SESSION_SECRET);
  if (payload.kind !== "sushiro_cloud_session" || !payload.login) {
    throw httpError(401, "invalid session");
  }
  if (!isAllowedGitHubUser(payload.login, env.ALLOWED_GITHUB_LOGINS || "")) {
    throw httpError(403, "GitHub account is not allowed");
  }
  return payload;
}

async function buildQueueBaselineExport(env, storeID) {
  const [stores, latestResult, rollupResult] = await Promise.all([
    fetchStores(env, storeID),
    fetchLatest(env, storeID),
    fetchRollups(env, storeID),
  ]);
  let sourceUpdatedAt = "";
  for (const item of latestResult.rows) if (item.collected_at > sourceUpdatedAt) sourceUpdatedAt = item.collected_at;
  if (rollupResult.sourceUpdatedAt > sourceUpdatedAt) sourceUpdatedAt = rollupResult.sourceUpdatedAt;
  return {
    version: 1,
    generated_at: new Date().toISOString(),
    source: "turso-cloudflare",
    bucket_minutes: DEFAULT_BUCKET_MINUTES,
    date_types: DATE_TYPES,
    stores,
    latest: latestResult.rows,
    rollups: rollupResult.rows,
    stats: {
      store_count: stores.length || (latestResult.rows.length || rollupResult.rows.length ? 1 : 0),
      latest_count: latestResult.rows.length,
      rollup_count: rollupResult.rows.length,
      source_updated_at: sourceUpdatedAt,
    },
  };
}

async function fetchStores(env, storeID) {
  const where = storeID ? `is_active = 1 AND store_id = ${storeID}` : "is_active = 1";
  const result = await tursoQuery(env, `SELECT
    store_id, name, city, area, address, latitude, longitude, open_date,
    tables_capacity, counters_capacity, last_seen_at
  FROM store_dimension
  WHERE ${where}
  ORDER BY store_id`);
  return result.rows.map((row) => ({
    store_id: asInt(row[0]),
    name: asString(row[1]),
    city: asString(row[2]),
    area: asString(row[3]),
    address: asString(row[4]),
    latitude: asNullableNumber(row[5]),
    longitude: asNullableNumber(row[6]),
    open_date: asString(row[7]),
    tables_capacity: asInt(row[8]),
    counters_capacity: asInt(row[9]),
    last_seen_at: asString(row[10]),
  }));
}

async function fetchLatest(env, storeID) {
  try {
    return await fetchLatestColumns(env, storeID, true);
  } catch (err) {
    if (!isMissingColumnError(err)) throw err;
    return fetchLatestColumns(env, storeID, false);
  }
}

async function fetchLatestColumns(env, storeID, includeCalled) {
  const where = storeID ? `WHERE store_id = ${storeID}` : "";
  const called = includeCalled ? ", display_called_no, group_queues_json" : "";
  const result = await tursoQuery(env, `SELECT
    store_id, collected_at, name, city, area, wait_minutes, group_queues_count,
    store_status, net_ticket_status, reservation_status, online_open,
    wait_time_counter, wait_time_cap${called}
  FROM store_latest
  ${where}
  ORDER BY store_id`);
  const rows = result.rows.map((row) => {
    const item = {
      store_id: asInt(row[0]),
      collected_at: asString(row[1]),
      name: asString(row[2]),
      city: asString(row[3]),
      area: asString(row[4]),
      wait_minutes: asInt(row[5]),
      group_queues_count: asInt(row[6]),
      store_status: asString(row[7]),
      net_ticket_status: asString(row[8]),
      reservation_status: asString(row[9]),
      online_open: asInt(row[10]) > 0,
      wait_time_counter: asInt(row[11]),
      wait_time_cap: asInt(row[12]),
    };
    if (includeCalled) {
      item.display_called_no = asInt(row[13]);
      item.group_queues_json = asString(row[14]);
    }
    return item;
  });
  return { rows };
}

async function fetchRollups(env, storeID) {
  try {
    return await fetchRollupColumns(env, storeID, true);
  } catch (err) {
    if (!isMissingColumnError(err)) throw err;
    return fetchRollupColumns(env, storeID, false);
  }
}

async function fetchRollupColumns(env, storeID, includeCalled) {
  const where = storeID ? `WHERE store_id = ${storeID}` : "";
  const called = includeCalled ? ", called_sample_count, called_no_slow, called_no_typical, called_no_fast" : "";
  const result = await tursoQuery(env, `SELECT
    store_id, date_type, weekday, time_bucket, sample_count, open_rate,
    online_open_rate, busy_rate, wait_typical_minutes, wait_safe_minutes,
    wait_max_minutes, queue_groups_typical, queue_groups_safe${called}, confidence, updated_at
  FROM store_bucket_rollups
  ${where}
  ORDER BY store_id, date_type, weekday, time_bucket`);
  let sourceUpdatedAt = "";
  const rows = result.rows.map((row) => {
    const confidenceIndex = includeCalled ? 17 : 13;
    const updatedAtIndex = includeCalled ? 18 : 14;
    const item = {
      store_id: asInt(row[0]),
      date_type: asString(row[1]),
      weekday: asInt(row[2]),
      time_bucket: asString(row[3]),
      sample_count: asInt(row[4]),
      open_rate: asNumber(row[5]),
      online_open_rate: asNumber(row[6]),
      busy_rate: asNumber(row[7]),
      wait_typical_minutes: asNullableNumber(row[8]),
      wait_safe_minutes: asNullableNumber(row[9]),
      wait_max_minutes: asInt(row[10]),
      queue_groups_typical: asNullableNumber(row[11]),
      queue_groups_safe: asNullableNumber(row[12]),
      confidence: asString(row[confidenceIndex]),
      updated_at: asString(row[updatedAtIndex]),
    };
    if (includeCalled) {
      item.called_sample_count = asInt(row[13]);
      item.called_no_slow = asNullableNumber(row[14]);
      item.called_no_typical = asNullableNumber(row[15]);
      item.called_no_fast = asNullableNumber(row[16]);
    }
    if (item.updated_at > sourceUpdatedAt) sourceUpdatedAt = item.updated_at;
    return item;
  });
  return { rows, sourceUpdatedAt };
}

async function tursoQuery(env, sql) {
  const url = tursoPipelineURL(env.TURSO_DATABASE_URL);
  const resp = await fetch(url, {
    method: "POST",
    headers: {
      authorization: `Bearer ${env.TURSO_AUTH_TOKEN}`,
      "content-type": "application/json",
      "x-libsql-client-version": "sushiro-cloud-worker",
    },
    body: JSON.stringify({
      requests: [{ type: "execute", stmt: { sql, want_rows: true } }],
    }),
  });
  const text = await resp.text();
  if (!resp.ok) throw httpError(502, `Turso HTTP ${resp.status}: ${text.slice(0, 400)}`);
  const data = JSON.parse(text);
  const result = data.results && data.results[0];
  if (!result) throw httpError(502, "Turso returned empty response");
  if (result.error) throw httpError(502, `${result.error.code || "TURSO_ERROR"}: ${result.error.message || ""}`);
  if (!result.response || result.response.type !== "execute") throw httpError(502, "Turso returned unexpected response");
  return result.response.result || { cols: [], rows: [] };
}

function tursoPipelineURL(raw) {
  // WHATWG URL 禁止把非特殊协议（libsql:）改写成特殊协议（https:），
  // u.protocol = "https:" 会静默失败，必须在解析前做字符串替换。
  let normalized = String(raw || "").trim();
  if (normalized.startsWith("libsql://")) normalized = "https://" + normalized.slice("libsql://".length);
  const u = new URL(normalized);
  if (u.protocol !== "https:" && u.protocol !== "http:") {
    throw httpError(500, `unsupported Turso URL scheme: ${u.protocol}`);
  }
  u.search = "";
  u.hash = "";
  const base = u.toString().replace(/\/+$/, "");
  return `${base}/v2/pipeline`;
}

function asString(value) {
  if (!value || value.type === "null" || value.value == null) return "";
  if (typeof value.value === "string") return value.value;
  return String(value.value);
}

function asInt(value) {
  const n = Number(asString(value));
  return Number.isFinite(n) ? Math.trunc(n) : 0;
}

function asNumber(value) {
  const n = Number(asString(value));
  return Number.isFinite(n) ? n : 0;
}

function asNullableNumber(value) {
  if (!value || value.type === "null" || value.value == null || value.value === "") return undefined;
  const n = Number(asString(value));
  return Number.isFinite(n) ? n : undefined;
}

function isMissingColumnError(err) {
  return /no such column|SQLITE_ERROR/i.test(String((err && err.message) || err));
}

async function signToken(payload, secret, ttlSeconds) {
  const now = nowSeconds();
  const body = { ...payload, iat: now, exp: now + ttlSeconds };
  const head = base64urlEncode(JSON.stringify({ alg: "HS256", typ: "JWT" }));
  const claims = base64urlEncode(JSON.stringify(body));
  const data = `${head}.${claims}`;
  const sig = await hmacSHA256(data, secret);
  return `${data}.${base64urlEncode(sig)}`;
}

async function verifyToken(token, secret) {
  const parts = String(token || "").split(".");
  if (parts.length !== 3) throw httpError(401, "invalid token");
  const data = `${parts[0]}.${parts[1]}`;
  const expected = base64urlEncode(await hmacSHA256(data, secret));
  if (!safeEqual(expected, parts[2])) throw httpError(401, "invalid token signature");
  const payload = JSON.parse(new TextDecoder().decode(base64urlDecode(parts[1])));
  if (!payload.exp || nowSeconds() >= Number(payload.exp)) throw httpError(401, "token expired");
  return payload;
}

async function hmacSHA256(data, secret) {
  const key = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  return new Uint8Array(await crypto.subtle.sign("HMAC", key, new TextEncoder().encode(data)));
}

function base64urlEncode(value) {
  const bytes = typeof value === "string" ? new TextEncoder().encode(value) : value;
  let binary = "";
  for (const b of bytes) binary += String.fromCharCode(b);
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

function base64urlDecode(value) {
  let s = String(value || "").replace(/-/g, "+").replace(/_/g, "/");
  while (s.length % 4) s += "=";
  const binary = atob(s);
  const out = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) out[i] = binary.charCodeAt(i);
  return out;
}

function safeEqual(a, b) {
  const aa = String(a || "");
  const bb = String(b || "");
  if (aa.length !== bb.length) return false;
  let diff = 0;
  for (let i = 0; i < aa.length; i++) diff |= aa.charCodeAt(i) ^ bb.charCodeAt(i);
  return diff === 0;
}

function workerCallbackURL(request) {
  const u = new URL(request.url);
  return `${u.origin}/auth/github/callback`;
}

function isAllowedLoopbackReturn(raw) {
  try {
    const u = new URL(raw);
    return u.protocol === "http:" && (u.hostname === "127.0.0.1" || u.hostname === "::1" || u.hostname === "localhost") && u.pathname === "/api/cloud/auth/callback";
  } catch (_) {
    return false;
  }
}

// fail-closed: 当 ALLOWED_GITHUB_LOGINS 为空时默认拒绝所有登录。
// 部署者必须设置 ALLOWED_GITHUB_LOGINS secret（英文逗号分隔的 GitHub login），
// 否则登录回调与 requireSession 都会拒绝访问，防止任意 GitHub 账号导出全国基准数据。
function isAllowedGitHubUser(login, allowlist) {
  const value = String(allowlist || "").trim();
  if (!value) return false;
  const allowed = new Set(value.split(",").map((x) => x.trim().toLowerCase()).filter(Boolean));
  return allowed.has(String(login || "").toLowerCase());
}

function parsePositiveInt(value) {
  const n = Number.parseInt(String(value || "").trim(), 10);
  return Number.isFinite(n) && n > 0 ? n : 0;
}

function nowSeconds() {
  return Math.floor(Date.now() / 1000);
}

function json(data, status = 200) {
  return new Response(JSON.stringify(data), { status, headers: JSON_HEADERS });
}

function httpError(status, message) {
  const err = new Error(message);
  err.status = status;
  return err;
}

function assertRequiredEnv(env) {
  const missing = REQUIRED_SECRETS.filter((key) => !String(env[key] || "").trim());
  if (missing.length) throw httpError(500, `missing Worker secrets: ${missing.join(", ")}`);
}
