# Contributing to the QBitFlow Go SDK

Thanks for helping. Bug reports and pull requests are welcome on
[GitHub](https://github.com/QBitFlow/qbitflow-go-sdk). For a security issue, don't open an issue:
follow [SECURITY.md](SECURITY.md).

## Requirements

Go 1.27 or later (the SDK uses generic methods). No other tool is needed for the gates below.

## Gates you can run

Run them from the repository root before opening a pull request; they need no network, no API key
and no QBitFlow account, and CI runs the same ones:

```bash
go build ./...
go vet ./...
go vet -tags integration ./...
test -z "$($(go env GOROOT)/bin/gofmt -l .)"   # the toolchain's gofmt (a PATH gofmt older than 1.27 skips generic methods)
go test -race -count=1 ./...
```

`go test ./...` runs the offline suite against in-process stub servers. Unset any `QBITFLOW_*`
variables you may have exported so the suite stays offline.

Two checks are skipped automatically on your machine; you don't need them for a pull request:

- **The snippet check** (`TestSnippets`) is part of the maintainers' conformance suite: it
  compares the examples' snippet regions and the README's embedded blocks with the website's
  snippet catalog, and skips outside the maintainers' workspace.
- **The live checks** (`integration_test.go`, build tag `integration`) run only with
  `QBITFLOW_API_KEY` and `QBITFLOW_BASE_URL` set. If you have a test-mode key, you may run the
  read-only group: `go test -tags integration -run LiveReadOnly -v .` ([Testing](README.md#testing)
  describes the write group).

## Code

- Keep the public surface documented: every exported identifier has a doc comment.
- Add or update tests with every change; behaviour is pinned by the offline suite.
- No new dependencies outside the standard library.
- Update the [README](README.md) when behaviour changes, and describe the change in the pull
  request (maintainers write the [CHANGELOG](CHANGELOG.md) entry).

## Website snippets

The code on [qbitflow.app/docs](https://qbitflow.app/docs) and in the README's Go blocks comes from
regions of the programs in [`examples/`](examples):

- **Regions.** A region is the code between two marker lines, exactly `// docs:start <id>` and
  `// docs:end <id>`. One region per id; regions never overlap or nest. The ids are shared with
  the website and the other QBitFlow SDKs: never rename or remove one.
- **Regions read on their own.** They assume a ready `*qbitflow.Client` named `client` and a `ctx`
  (except `client-init` and `client-on-behalf-of`); an id they need but don't create comes from a
  variable defined just outside the region (`paymentUUID`, `subscriptionUUID`, `sessionUUID`,
  `memberUUID`). They use no real keys (those come from `QBITFLOW_API_KEY` and
  `QBITFLOW_WEBHOOK_SECRET`), and handle errors with `log.Fatal(err)`.
- **The programs stay runnable** and build with the gates.
- **The manifest**, [`examples/snippets.manifest.json`](examples/snippets.manifest.json), maps each
  id to the file holding its region, or to the reason the SDK can't support it (`unsupported`).
- **The README** embeds regions between `<!-- docs:snippet <id> -->` and `<!-- /docs:snippet -->`
  lines. A maintainer's script fills them: never edit what is between them by hand. If your change
  edits a region, say so in the pull request and a maintainer regenerates the README blocks.
