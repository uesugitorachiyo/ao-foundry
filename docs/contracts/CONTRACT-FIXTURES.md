# Contract Fixtures

Each public schema has one valid fixture and one intentionally invalid fixture
under `examples/contract-fixtures/`.

The invalid fixtures are JSON documents that fail their target schema without
requiring malformed JSON. CI and release dry-run keep these files parseable so
schema test runners can consume them directly.

Run `go run ./cmd/foundry contract fixtures validate` to check every valid
fixture against its schema and confirm every invalid fixture is rejected by the
same schema.
