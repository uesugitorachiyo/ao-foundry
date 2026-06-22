# Clean-Clone Smoke

The public smoke path avoids sibling AO repositories and remote services.

```sh
go test ./...
go run ./cmd/foundry registry validate --registry examples/registry/clean-clone.foundry-registry.json
go run ./cmd/foundry task validate --task examples/tasks/clean-clone-smoke.foundry-task.json
go run ./cmd/foundry readiness audit --registry examples/registry/clean-clone.foundry-registry.json --task examples/tasks/clean-clone-smoke.foundry-task.json --out tmp/clean-clone-readiness.json
go run ./cmd/foundry release dry-run --out tmp/release-manifest.json
go run ./cmd/foundry release validate-manifest --manifest tmp/release-manifest.json
go run ./cmd/foundry contract fixtures validate
```

The smoke path is local-only. It does not require credentials, network access,
publishing, uploads, tags, pushes, or sibling repository checkouts.
