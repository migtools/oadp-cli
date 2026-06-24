# OADP CLI Claude plugin

Claude Code plugin that teaches assistants to use `oc oadp` / `kubectl oadp` for
OpenShift backup and restore.

Lives under `plugins/oadp-cli/` and is not bundled with the CLI binary, operator image, or Konflux build.

## Files

```
.claude-plugin/marketplace.json          # repo root — marketplace `oadp-cli-plugins`
plugins/oadp-cli/
├── .claude-plugin/plugin.json           # plugin manifest
├── skills/backup-restore/SKILL.md       # skill content — edit this
└── README.md
```

Marketplace entry points at `./plugins/oadp-cli` (see `marketplace.json`).

## Skill

- **Name:** `backup-restore`
- **Slash command:** `/oadp-cli:backup-restore`
- **Content:** [`skills/backup-restore/SKILL.md`](skills/backup-restore/SKILL.md)

When CLI commands or docs change, update the skill file and bump `version` in
`.claude-plugin/plugin.json`.

## Install

**GitHub marketplace** (after this is merged to `migtools/oadp-cli`):

```bash
claude plugin marketplace add github:migtools/oadp-cli
claude plugin install oadp-cli@oadp-cli-plugins
```

**Local clone** (while developing):

```bash
claude plugin marketplace add ~/git/oadp-cli
claude plugin install oadp-cli@oadp-cli-plugins
```

**Single session, no install:**

```bash
claude --plugin-dir ~/git/oadp-cli/plugins/oadp-cli
```

## Verify

```bash
claude plugin validate ~/git/oadp-cli/plugins/oadp-cli
claude plugin details oadp-cli
```

`plugin details` should show **Skills (1):** `backup-restore`.

In Claude Code: `/reload-plugins` or restart, then `/oadp-cli:backup-restore`.

After local edits, reload or reinstall if the cache is stale:

```bash
claude plugin install oadp-cli@oadp-cli-plugins
```

## Enterprise registration

| Field | Value |
|-------|-------|
| Marketplace | `github:migtools/oadp-cli` |
| Marketplace name | `oadp-cli-plugins` |
| Plugin | `oadp-cli@oadp-cli-plugins` |
| Manifest | `.claude-plugin/marketplace.json` |

## References

- [Claude Code plugins](https://code.claude.com/docs/en/plugins)
- [OADP CLI docs](https://github.com/migtools/oadp-cli/tree/main/docs)
