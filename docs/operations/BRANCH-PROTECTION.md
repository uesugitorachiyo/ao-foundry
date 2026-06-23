# Branch Protection

AO Foundry `main` is protected in GitHub and should require the current CI
matrix before merge:

- `test (ubuntu-latest)`
- `test (macos-latest)`
- `test (windows-latest)`

The branch protection policy also keeps strict status checks enabled, enforces
the rule for administrators, requires linear history, disables force pushes, and
disables branch deletion.

Verify the live policy with:

```sh
scripts/verify-branch-protection.sh
```

The verifier calls the GitHub branch protection read API and exits non-zero when
the live required checks or safety toggles drift from this document. It does not
push, merge, edit repository settings, or mutate branch protection.

The same verifier also runs from
`.github/workflows/production-readiness-ops.yml` on manual dispatch and a daily
schedule, using the repository-scoped `GITHUB_TOKEN` as `GH_TOKEN`.

When the token can read the full branch protection endpoint, the verifier runs in
`mode=full` and checks strict status checks, admin enforcement, linear history,
force-push protection, deletion protection, and required check names. When
GitHub Actions restricts the built-in token from that endpoint, the verifier
falls back to `mode=limited` through branch metadata and still checks that
`main` is protected for everyone with the required CI matrix contexts.
