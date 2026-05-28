# CLAUDE.md

Notes for contributors (and AI assistants) working on this provider.

## Public repo — privacy rules

This repo is **public** on GitHub and mirrored to the public Terraform Registry. Anything committed here is permanently indexable (GitHub archives, search caches, `git clone --mirror` by anyone).

**Never commit to this repo:**

- API tokens, deploy tokens, OAuth secrets, GPG keys, signing keys, kubeconfigs, `.terraform/` plugin cache contents, `.terraform.lock.hcl` from a private state, `terraform.tfstate*`.
- Real internal infrastructure identifiers: ServersCom dedicated-server IDs, live public IPv4 addresses owned by the account, internal hostnames, planned-but-unlaunched product domains.
- Internal company names, project codenames, ticket keys / project prefixes, Jira/Confluence URLs, internal chat channel names.
- Customer / operator email addresses, names, real account numbers, billing identifiers.

When writing this rule list itself, do not use a real-world value as the example of "what not to use" — describe categorically. The list above is itself an example: the offending text was concrete patterns that doubled as recon hints for anyone reading the public repo.

**Use in docs / examples instead:**

- Server IDs → `server-id` or `aBcDeFgH` placeholder.
- IPv4 → RFC 5737 documentation ranges: `192.0.2.0/24`, `198.51.100.0/24`, `203.0.113.0/24`.
- Domains → `example.com` and its subdomains (RFC 2606 reserved).
- PTR / record IDs → `recordId`, `ptrId` placeholders.

**If a private value lands in a commit anyway**: rotate it (token / key / etc.), then add a follow-up commit redacting the value. **Do not rewrite git history** — by the time you notice the leak it's already mirrored to GitHub clones, archive crawlers, and Terraform Registry; rewriting only breaks every existing clone without remediating the leak. Rotation is the only real fix.

## What this is

A Terraform provider that wraps Servers.com Public API endpoints missing from the official `serverscom/serverscom` provider. Built on `terraform-plugin-framework` v1.x. Published to the Terraform Registry as `AdconnectDevOps/serverscom-extras`.

Current surface: PTR records on dedicated servers. Other gap-fills (alias-IP add/remove if API exposes it, cloud-instance PTR, l2 advanced) belong here as separate resources, not as separate repos.

## Layout

```
.
├── main.go                                # provider entrypoint
├── provider.go                            # ServersComExtrasProvider — schema, Configure, resource list
├── serverscom/                            # provider package
│   ├── client.go                          # REST client + PtrRecord/PtrCreateRequest types
│   ├── rate_limiter.go                    # mutex-based request spacer + 429 retry
│   └── resource_ptr_record.go             # serverscom_ptr_record resource
├── docs/                                  # registry-published docs
├── examples/                              # runnable HCL examples
├── .goreleaser.yml                        # release build matrix
└── .github/workflows/                     # CI (test on PR, release on tag)
```

## Common commands

```bash
make build      # build for darwin/arm64
make install    # build + install to ~/.terraform.d/plugins/
make test
make fmt vet
make dev        # run with -debug for TF_REATTACH_PROVIDERS attach
```

## Release flow

1. Update `CHANGELOG.md` (move Unreleased → new version header with date).
2. `make vet && make build` locally.
3. `git tag vX.Y.Z && git push origin vX.Y.Z`.
4. GH Action runs goreleaser (linux/darwin × amd64/arm64), GPG-signs the checksum file, creates GitHub release.
5. Registry picks up within ~10-15 min:
   ```bash
   curl -s https://registry.terraform.io/v1/providers/AdconnectDevOps/serverscom-extras/versions \
     | python3 -c "import json,sys; print(sorted([v['version'] for v in json.load(sys.stdin)['versions']], key=lambda x:[int(p) for p in x.split('.')])[-3:])"
   ```

Required GitHub secrets: `GPG_PRIVATE_KEY` (ASCII-armored signing key). `GITHUB_TOKEN` is set automatically.

## Provider Framework conventions

Inherits from `terraform-provider-shodan`'s contributor guide. Key rules:

- **Computed `id` uses `UseStateForUnknown`** — without it, Terraform marks the value as `<known after apply>` on every plan and Update paths that don't reassign `plan.ID = state.ID` write empty to state, then Read 404s on the empty URL.
- **Update must carry forward state ID** — even with `UseStateForUnknown`, defensively assign `plan.ID = state.ID` before any API mutation in Update. (For `serverscom_ptr_record`, Update is a no-op — every attr is `RequiresReplace`.)
- **Read self-heals on 404** — `resp.State.RemoveResource(ctx)` and return; Terraform recreates on next plan.
- **Client method conventions** — empty-ID guard, rate-limited HTTP client (never `http.DefaultClient`), idempotent DELETE (treat 404 as success), wrap non-2xx as `fmt.Errorf("API request failed with status %d: %s", code, body)` so callers can string-match `status 404`.

## Adding a new resource

1. Create `serverscom/resource_<thing>.go` modelled on `resource_ptr_record.go`.
2. Add `NewXResource` to the slice returned by `Resources()` in `provider.go`.
3. Add user docs at `docs/resources/<thing>.md`.
4. Add a runnable example at `examples/resources/serverscom_<thing>/resource.tf`.
5. Update `CHANGELOG.md` under Unreleased.

## API quirks (observed via OPTIONS)

| Endpoint | Allowed methods | Notes |
|---|---|---|
| `/hosts/dedicated_servers/{id}/networks` | OPTIONS, GET, HEAD | Alias-IP ordering is portal-only, not via API. |
| `/hosts/dedicated_servers/{id}/ptr_records` | OPTIONS, GET, POST, HEAD | Collection — create + paginated list. |
| `/hosts/dedicated_servers/{id}/ptr_records/{id}` | OPTIONS, DELETE | No PUT/PATCH — PTR records are immutable post-create. |

The "no PUT" finding drives `RequiresReplace` on every PTR schema attribute. If Servers.com later adds PATCH/PUT, drop `RequiresReplace` on `domain`/`priority`/`ttl` and add proper Update logic — `host_id` and `ip` stay ForceNew (PTR identity).

## Auth + rate limits

- Token via `Authorization: Bearer <token>` header. Env fallback: `SERVERSCOM_TOKEN`.
- API rate limit advertised as 2000 req/min per token (`x-ratelimit-limit: 2000`). The 1-second floor in `rate_limiter.go` is conservative; bump if mass-import needs more throughput.

## Style

- `gofmt -s` everything (`make fmt`).
- Comments explain *why*, not *what*. Public methods get a single-line Go-style doc comment.
- No `panic` in non-fatal paths — surface via Diagnostics.
