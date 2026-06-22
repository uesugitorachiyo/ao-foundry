# Contributing

## Development

Run the public verification set before proposing changes:

```sh
go test ./...
jq empty docs/contracts/*.json examples/**/*.json
go run ./cmd/foundry competitive audit
```

## Boundaries

- Keep AO Foundry focused on engineering operations coordination above AO Forge.
- Do not add credential storage or remote publishing to normal verification.
- Do not add release automation that pushes, tags, uploads, or publishes.
- Keep examples public-safe and reproducible from a clean clone.
