# Repository Archive Hygiene

Use repository exports only for committed source state. Do not package a working
directory by zipping the folder manually: that can include `.git`, IDE state,
runtime logs, local request payloads, development certificates, and private keys.

Canonical export command from the repository root:

```powershell
.\scripts\export-archive.ps1
```

The default output is:

```text
.artifacts/pet-study-source.zip
```

The export script uses `git archive HEAD`, so the archive contains only tracked
files from the selected commit. It writes to a temporary ZIP first, validates
paths and content, then publishes the final output only after validation passes.
If validation fails, the temporary ZIP is removed and any existing final archive
is left unchanged.

Blocked archive content:

- `.git/`
- `.idea/`, `.vscode/`, `*.iml`, `*.swp`
- `.DS_Store`, `Thumbs.db`
- `.artifacts/`, `bin/`, `dist/`, `tmp/`
- runtime logs such as `server.log` or `*.log`
- request scratch files such as `req.json` and `req-async.json`
- accidental shell/curl scratch files such as `-H`
- local environment files such as `.env`, `.env.*`, `*.local`
- development certificates and key material under `certs/`, `*.pem`, `*.key`,
  `*.ppk`, `*.p12`, `*.pfx`

The content scan checks archive entries for PEM private-key markers using a
bounded streaming scan, including text entries larger than 1 MiB.

Development certificates and keys are local-only artifacts. Generate them on
each developer machine when HTTPS smoke testing is needed, and point
`HTTP_TLS_CERT_FILE` and `HTTP_TLS_KEY_FILE` at those local files. Never commit
or share generated private keys.

To overwrite an existing local export:

```powershell
.\scripts\export-archive.ps1 -Force
```
