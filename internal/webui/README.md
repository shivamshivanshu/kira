# Web dashboard

The dashboard is an embedded Go HTTP application. There is no frontend build,
package installation, or separate production runtime. Mutations call `core` or
the restricted CLI bridge; UI controls must not implement workflow rules.

## Structure

- `server.go`, `security.go`: HTTP routing, request security, and result handling.
- `page.go`, `viewmodel.go`: core-backed reads and template-facing data/helpers.
- `actions.go`, `management.go`, `bulk.go`: thin form-to-core adapters.
- `assets.go`, `static.go`: template/Markdown rendering and versioned static files.
- `templates/`: named partials by responsibility, composed by `page.html`.
- `static/css/`: theme tokens, base rules, layout, components, and work views.
- `static/js/`: focused native ES modules initialized by `main.mjs`.
- `testdata/`: dependency-free JavaScript tests; Go tests live beside the code.

## Extending the UI

Add colors to `static/css/tokens.css`, not individual component rules. Keep the
explicit and system dark palettes consistent. Use semantic tokens for status and
diff colors, and preserve reduced-motion behavior. Contrast and token-reference
tests guard these constraints.

Add markup to its feature partial. Give controls accessible labels and connect
them through a focused JavaScript module or an ordinary Go-backed form. Palette
actions delegate to the same controls. Keep query filters in the existing core
query machinery, not a second parser in the browser.

`appearance.js` is a small blocking script that applies saved preferences before
styles load, avoiding a light flash in dark mode. It intentionally has no module
dependencies. The runtime preferences module handles system changes and cross-tab
updates; tests check both paths. Other scripts use relative ES module imports.

Use the `asset` template helper for entrypoint URLs and relative paths for CSS and
JavaScript imports. A content fingerprint versions the entire static tree, so
assets can be cached immutably without serving stale code after a rebuild. HTML,
action results, and repository data remain uncached. Page loading avoids repeated
index reads through a request-scoped core snapshot and computes expensive data
only when the view needs it. Detail content and its edit revision come from the
same file read; commit previews reuse that detail's validated commit links.
Rendering and response writes run outside the store/notice mutex.

Command bars share `internal/commandline` tokenization, while the web bridge keeps
its own execution restrictions. Warning messages come from `internal/warningfmt`.
Item views supply `data-copy-values` through `pageData.CopyValues`; one shared
copy-picker template supplies the buttons and labels. The item picker searches a
compact JSON item list and creates at most 50 result links when opened or filtered.
Use the shared commit-card partial for commit views. Page rendering prepares its
query parameters, item-name lookup and bulk-selection set once.

## Verification

```sh
go test -race ./internal/webui ./internal/cli
node --test internal/webui/testdata/*_test.mjs
go test ./internal/webui -run '^$' -bench . -benchmem
```

The Go suite runs JavaScript state tests when Node is available. Benchmarks use
temporary seeded repositories, not the current checkout. For visual changes,
also check both themes, system switching, compact spacing, narrow layouts,
keyboard navigation, and reduced motion in a browser.
