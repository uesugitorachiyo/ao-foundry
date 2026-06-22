# Security Policy

## Supported Versions

AO Foundry v0.1 is a local prototype. Security reports should target the public
schemas, CLI behavior, and release hardening checks in this repository.

## Reporting

Report security issues through GitHub Security Advisories for this repository.
If advisories are unavailable, open a minimal public issue that describes the
affected public file or command without including secrets, tokens, exploit
payloads, or private operational material.

## Local Safety Model

- Commands are local-first by default.
- Release, upload, tag, push, and credential access are blocked unless a future
  explicit approval flow allows a bounded side effect.
- Public fixtures must not contain local absolute paths or credential-like
  strings.
