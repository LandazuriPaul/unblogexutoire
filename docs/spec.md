# Un blog exutoire — specification

Status: draft for review (2026-11-23). Decisions log: `docs/spec-grilling.md`.

## 1. Goals and non-goals

**Goals**

- A small, French-language, multi-author blog at `https://unblogexutoire.ninja`.
- Near-zero maintenance: upgrading means bumping Go and two or three modules.
- Minimal attack surface: no admin UI, no accounts, no database, no writable filesystem.
- No licence risk: only the Go standard library and permissively licensed modules.

**Non-goals**

- Comments, analytics, newsletter, client-side JavaScript, an editor UI, media uploads, i18n.

## 2. Architecture

- One Go program (`main` package plus a few internal packages). Dependencies:
  - `github.com/yuin/goldmark` for Markdown (safe mode: raw HTML is stripped)
  - a YAML parser for frontmatter and `authors.yaml`
  - `golang.org/x/text` to fold accents in search
- `content/`, `templates/` and `static/` are embedded with `embed.FS`.
- At startup the binary parses and validates all content, then pre-renders every static page
  (HTML plus a gzip version) into memory. It refuses to start if any content is invalid.
- It serves over `net/http` on `:8080`. The only page rendered per request is the posts page
  with search.
- Flags: `-addr` (default `:8080`), `-drafts` (include drafts, for local preview only) and
  `-base-url` (default `https://unblogexutoire.ninja`, used for the feed, sitemap and OG URLs).

## 3. Content model

```
content/
  authors.yaml          # list of {slug, name, bio?}
  posts/YYYY-MM-DD-<slug>.md
  pages/<slug>.md       # e.g. a-propos
```

Post frontmatter:

| Field         | Required | Notes                                                       |
|---------------|----------|-------------------------------------------------------------|
| `title`       | yes      |                                                             |
| `slug`        | yes      | `[a-z0-9-]+`, unique across posts and pages, not reserved   |
| `date`        | yes      | RFC 3339 or `YYYY-MM-DD`                                    |
| `authors`     | yes      | list of author slugs that exist in `authors.yaml`           |
| `tags`        | no       | list of display strings; slug derived (lowercase, accent-folded) |
| `description` | no       | fallback: first ~30 words of the body, as plain text        |
| `status`      | no       | `published` (default) or `draft`                            |

Pages use `title`, `slug` and optionally `description` and `status`.

Two tags with the same slug must have the same display name (so `Récit` and `Recit` is an
error). Unknown frontmatter keys are an error. Dates are shown and filtered in `Europe/Paris`.

Reserved slugs: `articles`, `author`, `tag`, `rss`, `static`, `healthz`, `sitemap.xml`, `robots.txt`.

**Migration:** move `backup/posts`, `backup/drafts` and `backup/pages` into `content/`. Drop
`ghost_id`, `created_at` and `updated_at`, turn author names into slugs, and create
`authors.yaml` with the 5 existing authors. Drafts get `status: draft`, and those without a date
get one. `backup/` stays git-ignored.

## 4. Routes

| Route                | Content                                                              |
|----------------------|----------------------------------------------------------------------|
| `/`                  | All published posts, newest first (title, date, authors, description) + link to `/articles/` |
| `/articles/`         | Search and list (see §5)                                             |
| `/{slug}/`           | Post or page                                                         |
| `/author/{slug}/`    | Author name, bio, all their posts                                    |
| `/tag/{slug}/`       | All posts with that tag                                              |
| `/rss/`              | Atom feed: all published posts, full content                         |
| `/sitemap.xml`, `/robots.txt` | Generated                                                   |
| `/static/style.<hash>.css` | Content-hashed CSS                                             |
| `/healthz`           | `200 ok`                                                             |
| anything else        | French 404 page                                                      |

Requests without a trailing slash are redirected to the slashed form (Ghost's behaviour).

**Post page:** title, authors (linked), long French date ("10 juin 2019"), tags (linked), body,
and « Article précédent / suivant » links. No reading time, no "updated on" date.

## 5. Posts page and search

- A GET form with no JS. Query parameters: `q`, `auteur` (author slug), `tag` (tag slug),
  `annee` and `mois`. Any other parameter (such as `page`, from before pagination was
  removed) is ignored.
- Dropdowns are filled from the data: the authors, the tags in use, and the years that have
  posts. The month dropdown is optional (1–12).
- Free text: the query and the documents are lowercased and accent-folded, then split on
  non-letters. A post matches if **every** query word appears as a substring of its folded
  title, description or body text.
- All filters are combined with AND. Every matching post is listed on one page, newest first.
  There is no pagination: the archive is small enough to scroll.
- The total result count is shown ("12 articles").

## 6. Presentation

- `html/template` with one base layout. A single hand-written CSS file of about 150 lines:
  serif body (system serif stack), ~65ch column, automatic dark mode via
  `prefers-color-scheme`. "Un blog exutoire" is the only branding.
- `<html lang="fr">`, plus Open Graph `og:title`, `og:description`, `og:type` and `og:url`, and
  `<meta name="description">`.

## 7. HTTP behaviour

- Headers on every response: `Content-Security-Policy: default-src 'self'`,
  `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`.
- Pre-rendered HTML: `Cache-Control: public, max-age=300` plus a strong ETag (supports 304). gzip
  is served when the client accepts it.
- Hashed CSS: `Cache-Control: public, max-age=31536000, immutable`.
- Search responses: `Cache-Control: public, max-age=60`, compressed on the fly.

## 8. Quality

- `go test ./...` covers content validation over the real `content/` directory, search matching
  (accents, AND, filters), routes (status codes, redirects, 404) and the feed.
- `go vet` and `gofmt` run in CI.

## 9. Tooling (mise)

Every tool and every command goes through [mise](https://mise.jdx.dev), the same way locally,
in CI and in the Docker build (the same layout as the maintainer's other Go projects):

```
.miserc.toml          # env = ["dev"]: local default loads the dev tools on top of the base
mise/config.toml      # base: go = "1.27.4" + all tasks           -> local, CI, Docker build
mise/config.dev.toml  # "go:golang.org/x/tools/gopls" (for Zed)    -> local only
mise/mise.lock        # locked with `mise lock --platform linux-x64,linux-arm64,macos-arm64`
mise/mise.dev.lock
mise/tasks/image/*.sh # file tasks for the Docker image
```

Tasks (`mise run <task>`, `mise tasks` lists them):

| Task          | Does                                                                        |
|---------------|-----------------------------------------------------------------------------|
| `dev`         | `go run . -drafts` on `:8080` (local preview including drafts)              |
| `serve`       | `go run .` (what production serves)                                         |
| `vet`         | `gofmt -l` must print nothing, then `go vet ./...`                          |
| `test`        | `go test ./...`                                                             |
| `image:build` | `docker build`, tagged `ghcr.io/landazuripaul/unblogexutoire:<short-sha>` and `:latest` |
| `image:run`   | builds, then runs the image on `:8080` with the production flags            |
| `image:push`  | builds, then pushes both tags (needs `docker login ghcr.io`)                |

The Go version is **only** pinned in `mise/config.toml` (and the matching `go` line in `go.mod`).
Upgrading Go is a manual change to both, then `mise lock`, a few times a year. Dependabot
doesn't read mise configs.

## 10. Build and CI

- Multi-stage `Dockerfile`. The builder is `ghcr.io/jdx/mise:<pinned>-debian` with
  `MISE_ENV=""` (base config only), `mise install --locked`, then `go build` with
  `CGO_ENABLED=0`. The runtime is `gcr.io/distroless/static-debian12:nonroot`, listening on 8080.
- GitHub Actions (`jdx/mise-action`, `MISE_ENV=""`): `mise run vet` and `mise run test` on
  PRs and on `main`. On push to `main` it also runs `docker login ghcr.io` with `GITHUB_TOKEN`
  and then `mise run image:push`, which pushes to **private**
  `ghcr.io/landazuripaul/unblogexutoire:<short-sha>` and `:latest`. The runner is amd64, so is
  the image. The VPS (raichu, OVH) is assumed to be amd64; check this before the first deploy.
- Dependabot: monthly, grouped, for `gomod`, `docker` (the mise builder and distroless images)
  and `github-actions`. No auto-merge.

## 11. Deployment

- Manifests live in the infra repo at
  `pokedex/devices/raichu/k3s/unblogexutoire/`: Namespace, Deployment, Service, a Traefik
  IngressRoute and a `www` → apex redirect Middleware. They're already drafted.
- TLS comes from Traefik's ACME resolver, the default on `websecure`. DNS is behind Cloudflare.
- The pod runs as non-root (65532), with a read-only root filesystem, all capabilities dropped
  and no service-account token. Resources: 10m CPU and 16Mi requested, 64Mi limit. A `/healthz`
  probe is set.
- **Manual release:** set the new short-SHA tag in `deployment.yaml`, commit it, and run
  `kubectl apply`.
- One-off step: `NAMESPACES=unblogexutoire patch-service-accounts.sh` for the private GHCR pull.

## 12. Contribution workflow

Authors send text by any channel. The maintainer writes `content/posts/<date>-<slug>.md`,
previews it with `mise run dev`, and pushes to `main`. CI builds the image, and the
maintainer bumps the tag in the infra repo. Authors comfortable with git can open a PR instead.
