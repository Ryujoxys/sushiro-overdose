import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const source = await readFile(new URL('../src/index.js', import.meta.url), 'utf8');
const { default: worker } = await import('data:text/javascript;base64,' + Buffer.from(source).toString('base64'));

test('all former cloud routes are gone without redirects or upstream calls', async () => {
  const original = globalThis.fetch;
  let calls = 0;
  globalThis.fetch = async () => { calls++; throw new Error('unexpected network call'); };
  try {
    for (const path of ['/', '/auth/github', '/auth/github/callback?code=secret', '/api/me', '/api/baseline']) {
      for (const method of ['GET', 'POST']) {
        const response = await worker.fetch(new Request('https://example.invalid' + path, { method }), {
          TURSO_AUTH_TOKEN: 'old-token', GITHUB_CLIENT_SECRET: 'old-secret',
        });
        assert.equal(response.status, 410);
        assert.equal(response.headers.get('location'), null);
        assert.equal(response.headers.get('cache-control'), 'no-store');
        assert.equal((await response.json()).code, 'cloud_data_retired');
      }
    }
    assert.equal(calls, 0);
  } finally {
    globalThis.fetch = original;
  }
});
