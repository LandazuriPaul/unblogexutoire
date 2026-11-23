# Un Blog Exutoire

This repository is a simple project to allow a small group of authors to share random blog posts.

It's one Go binary that serves [unblogexutoire.ninja](https://unblogexutoire.ninja). The posts are
Markdown files in `content/`, embedded into the binary at build time. There's no database, no admin
UI and no JavaScript. The spec lives in [`docs/spec.md`](docs/spec.md).

## Prerequisites

- [mise](https://mise.jdx.dev): installs Go (and gopls for your editor) at the pinned versions
- Docker, only for the image tasks

```sh
mise install
```

## Tasks

Every command goes through mise, locally and in CI (`mise tasks` lists them):

| Task               | Does                                                                     |
|--------------------|--------------------------------------------------------------------------|
| `mise run dev`     | Serve on http://localhost:8080, **drafts included**                      |
| `mise run serve`   | Serve on :8080 exactly as production does (no drafts)                    |
| `mise run vet`     | `gofmt` check and `go vet`                                               |
| `mise run test`    | All tests, including a check of every file in `content/`                 |
| `mise run image:build` | Build `ghcr.io/landazuripaul/unblogexutoire:<short-sha>` and `:latest` (`-dirty` if the tree has changes) |
| `mise run image:run`   | Build, then run the image on :8080, read-only, like the cluster does  |
| `mise run image:push`  | Build, then push both tags (needs `docker login ghcr.io`; refuses `-dirty`) |

## Adding a post

1. Create `content/posts/YYYY-MM-DD-<slug>.md`:

   ```markdown
   ---
   title: "Le titre du billet"
   slug: le-titre-du-billet
   date: 2026-11-23
   authors:
     - la-choucroute-masquee
   tags:
     - "récit"
   description: "Optional one-line summary; otherwise the first 30 words are used."
   status: draft   # remove this line to publish
   ---

   Le texte, en Markdown.
   ```

   - `slug`: lowercase letters, digits and dashes. It becomes the URL (`/le-titre-du-billet/`).
   - `authors`: slugs from [`content/authors.yaml`](content/authors.yaml). Add a new author there first.
   - `tags`: free text. Spell a tag the same way every time: `Récit` and `récit` are different
     spellings of the same tag, and the tests reject that.
   - `date`: `YYYY-MM-DD` or a full RFC 3339 timestamp. Dates are shown in Paris time.
   - Raw HTML in the body is stripped.

2. Preview it with `mise run dev`. Then run `mise run test`, which fails with the file name and the
   reason if anything is wrong.
3. Commit and push to `main`.

## Releasing

Every push to `main` runs the checks, then pushes `ghcr.io/landazuripaul/unblogexutoire:<short-sha>`
(a private package). Deploying is manual: set that tag in
`pokedex/devices/raichu/k3s/unblogexutoire/deployment.yaml`, commit it, and
`kubectl apply -f deployment.yaml`.

## Upgrading

- Go modules, the Docker base images and the GitHub Actions: Dependabot opens one grouped PR a
  month. Merge it once CI is green.
- Go itself is pinned in `mise/config.toml`, which Dependabot doesn't read. Bump `go` there and the
  `go` line in `go.mod`, then run:

  ```sh
  mise lock --platform linux-x64,linux-arm64,macos-arm64
  mise run test
  ```

`mise/config.toml` sets `GOTOOLCHAIN=local`, so the Go that mise installs is always the one used,
even if a global `go env -w GOTOOLCHAIN=…` would otherwise make Go download and switch to
another toolchain.
