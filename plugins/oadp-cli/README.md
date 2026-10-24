# OADP CLI Claude plugin

Claude Code plugin that teaches assistants to use `oc oadp` / `kubectl oadp` for OpenShift backup and restore.

Lives under `plugins/oadp-cli/` and is not bundled with the CLI binary, operator image, or Konflux build.

## Files

```text
.claude-plugin/marketplace.json          # repo root — Claude marketplace `oadp-cli-plugins`
plugins/oadp-cli/
├── .claude-plugin/plugin.json           # plugin manifest
├── skills/backup-restore/SKILL.md       # skill content — edit this
└── README.md
```

The Claude marketplace manifest points at `./plugins/oadp-cli` (see `marketplace.json`).

## Skill

- **Name:** `backup-restore`
- **Slash command:** `/oadp-cli:backup-restore`
- **Content:** [`skills/backup-restore/SKILL.md`](skills/backup-restore/SKILL.md)

When CLI commands or docs change, update the skill file and bump `version` in `.claude-plugin/plugin.json`.

## Customer usage

### 1. Install the plugin

The plugin is would be available in the Claude community marketplace. Add the marketplace once, then install:

```bash
claude plugin marketplace add anthropics/claude-plugins-community
claude plugin install oadp-cli@claude-community
```

This only needs to be done once. The plugin persists across Claude Code sessions.

Or install from inside an active Claude Code session:

```text
/plugin marketplace add anthropics/claude-plugins-community
/plugin install oadp-cli@claude-community
```

### 2. Start Claude Code

```bash
claude
```

If the plugin was just installed in an active session, run `/reload-plugins` to pick it up without restarting.

### 3. Use the skill

Invoke explicitly:

```text
/oadp-cli:backup-restore
```

Or just ask naturally since the skill triggers automatically for OADP questions:

```text
How do I back up the my-app namespace with OADP?
How do I restore a namespace using the non-admin workflow?
How do I set up a backup storage location?
```

Claude will respond using `oc oadp` / `kubectl oadp` commands rather than raw CRD edits.

### 4. Uninstall

```bash
claude plugin uninstall oadp-cli@claude-community
```

## Developer install

For contributors or local development before marketplace availability:

**From a local clone:**

```bash
claude plugin marketplace add <path-to-oadp-cli>
claude plugin install oadp-cli@oadp-cli-plugins
```

**Single session, no install:**

```bash
claude --plugin-dir <path-to-oadp-cli>/plugins/oadp-cli
```

## Verify

```bash
claude plugin validate <path-to-oadp-cli>/plugins/oadp-cli
claude plugin details oadp-cli
```

`plugin details` should show Skills (1): `backup-restore`.

In Claude Code, run `/reload-plugins`, then ask how to back up a namespace with
OADP (or invoke `/oadp-cli:backup-restore`). Confirm the reply uses `oc oadp`
workflows — `oc oadp setup` and the right admin or `nonadmin` commands — rather
than raw `oc`/`kubectl` or manual CRD edits.

After local edits, reload or reinstall if the cache is stale:

```bash
claude plugin install oadp-cli@oadp-cli-plugins
```

## Enterprise registration

| Field | Value |
|-------|-------|
| Claude community marketplace | `claude-community` |
| Marketplace source | `anthropics/claude-plugins-community` |
| Plugin | `oadp-cli@claude-community` |

## References

- [Claude Code plugins](https://code.claude.com/docs/en/plugins)
- [Claude community marketplace](https://code.claude.com/docs/en/discover-plugins)
- [OADP CLI docs](https://github.com/migtools/oadp-cli/tree/oadp-dev/docs)
