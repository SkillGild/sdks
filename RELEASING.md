# Releasing

Each SDK is versioned independently with semantic versioning and released from a tag.
Keep all three at `0.x` until the public REST contract has a reviewed compatibility
policy; a `0.x` release may change APIs between minor versions.

| SDK | Registry | Tag |
| --- | --- | --- |
| TypeScript | npm `@skillgild/sdk` | `typescript/v0.1.0` |
| Python | PyPI `skillgild` | `python/v0.1.0` |
| Go | `github.com/skillgild/sdks/go` (proxy.golang.org) | `go/v0.1.0` |

## Steps

1. Bump the version (`package.json`, `pyproject.toml` and `skillgild/__init__.py`; Go
   uses the tag alone) and add the entry to that SDK's `CHANGELOG.md` in the same commit.
2. Merge to `main` with CI green.
3. Push the tag, for example `git tag typescript/v0.1.1 && git push origin typescript/v0.1.1`.
   The `release` workflow tests and publishes it. Go needs no publish step: the proxy
   fetches the tag on first `go get`, and the workflow warms it.

Publishing uses npm and PyPI **trusted publishing** (OIDC); no registry token is
stored in this repository.

## Compatibility rules

- Adding a response field or a method is compatible. Removing or renaming one is breaking.
- SDKs never bundle skill prompts, credentials or server implementation details.
- Before a release, check behaviour against the [API reference](https://skillgild.dev/docs/api):
  auth errors, quota responses, URL validation and idempotency.
