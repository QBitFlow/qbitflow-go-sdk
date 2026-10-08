# Contributing to the QBitFlow Go SDK

## Gates

Run them from the repo root before committing; all must pass, offline (unset the `QBITFLOW_*`
variables):

```bash
go build ./...
go vet ./...
test -z "$($(go env GOROOT)/bin/gofmt -l .)"   # the toolchain's gofmt, not the one on PATH
go test -count=1 ./...
go test -race -count=1 ./...
```

The live checks are opt-in: see [Testing](README.md#testing).

## Website snippets

> **Website snippets.** The code on qbitflow.app/docs and in this README comes from `// docs:start <id>` … `// docs:end <id>` regions in `examples/`. Every change keeps them true:
> - **New public capability:** add a region, an id in the hub's `snippets/catalog.json` (the same id across all SDKs), and the manifest entry, plus the README block via `scripts/embed-snippets.mjs`.
> - **Changed capability:** update the region's code. Never hand-edit an embedded README block: re-run the script.
> - **Ids are a public contract with the website.** Never rename or remove one silently. To retire an id, keep it until the website no longer uses it, and note it in the CHANGELOG.
> - **Before committing:** the examples run, the manifest check passes, and `embed-snippets --check` passes.
> - **Unsupported ids:** an id the SDK can't support goes under `unsupported` with a reason. Never write placeholder code.

### The convention

- **Regions.** In a program under `examples/`, wrap the code the docs show between two marker
  lines, exactly `// docs:start <id>` and `// docs:end <id>` (one space each side; indentation
  before them is fine, the extractor dedents). One region per id; regions never overlap or nest.
  The ids come from the hub's `snippets/catalog.json`, whose `shows` and `routes` say what each
  region does.
- **Regions read on their own.** They assume a ready `*qbitflow.Client` named `client` and a
  `ctx` (except `client-init` and `client-on-behalf-of`); an id they need but don't create comes
  from a variable defined just outside the region (`paymentUUID`, `subscriptionUUID`,
  `sessionUUID`, `memberUUID`). They use the catalog's shared example values (`conventions`), the
  `qbitflow.PlaceholderUUID` constant in success URLs, and no real keys: those come from
  `QBITFLOW_API_KEY` and `QBITFLOW_WEBHOOK_SECRET`. Setup (env lookups, flags, main wiring) stays
  outside. Errors: `log.Fatal(err)`, as in the rest of the examples.
- **The programs stay runnable**: each one reads its configuration from the environment (see
  [Examples](README.md#examples)) and builds with the gates.
- **The manifest**, `examples/snippets.manifest.json`, maps every catalog id to the file holding
  its region (`snippets`), or to the reason this SDK can't support it (`unsupported`), in the
  catalog's order, and pins the hub commit of the catalog it follows (`catalog`).
- **The README** embeds regions between `<!-- docs:snippet <id> -->` and
  `<!-- /docs:snippet -->` lines. The script fills them: never edit what is between them.

### Running the script

From a hub checkout (this repo sits in its `sdk/qbitflow-go-sdk`):

```bash
node ../../scripts/embed-snippets.mjs .          # rewrites the README's embedded blocks
node ../../scripts/embed-snippets.mjs --check .  # changes nothing; exits 1 on any drift
```

With the hub elsewhere: `node "$QBITFLOW_HUB_DIR/scripts/embed-snippets.mjs" --check .`.

`go test ./...` runs the same check (`TestSnippets` in `snippets_test.go`). It finds the hub at
`QBITFLOW_HUB_DIR` when set (and fails if that is not a hub), else at `../..`; without a hub or
`node`, it is skipped.
