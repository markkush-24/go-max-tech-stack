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
files from the selected commit. It also rejects blocked paths and scans small
archive entries for PEM private-key markers before reporting success.

Blocked archive content:

- `.git/`
- `.idea/`, `.vscode/`, `*.iml`
- `.artifacts/`, `bin/`, `dist/`, `tmp/`
- runtime logs such as `server.log` or `*.log`
- request scratch files such as `req.json` and `req-async.json`
- accidental shell/curl scratch files such as `-H`
- local environment files such as `.env`, `.env.*`, `*.local`
- development certificates and key material under `certs/`, `*.pem`, `*.key`

Development certificates and keys are local-only artifacts. Generate them on
each developer machine when HTTPS smoke testing is needed, and point
`HTTP_TLS_CERT_FILE` and `HTTP_TLS_KEY_FILE` at those local files. Never commit
or share generated private keys.

To overwrite an existing local export:

```powershell
.\scripts\export-archive.ps1 -Force
```
