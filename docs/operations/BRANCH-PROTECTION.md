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
