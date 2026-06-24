# OADP CLI Claude plugin

Part of the **OADP CLI Awareness Initiative**: encourage OpenShift customers to
use **`oc oadp`** / **`kubectl oadp`** for backup and restore via AI assistants.

This plugin is **not** part of the official `oadp-cli` binary, operator image, or
Konflux payload. It is a Claude skill only, which avoids AI-product audit
requirements on the shipped CLI.

## Initiative context

| Track | What | This plugin |
|-------|------|-------------|
| **OpenShift official support for OADP** | Product: `oc oadp`, operator, downstream images on `registry.redhat.io` | Complements the product; does not ship inside it |
| **Claude plugin for oadp-cli** | Skill + registration on enterprise Claude | **This repo** — primary deliverable |
| **OpenShift Lightspeed + MCP** | Lightspeed deploys the OpenShift MCP server | Inherits this plugin's CLI guidance via MCP tools/prompts |

```
OpenShift official support for OADP          (product — no AI)
        │
        ├── Claude plugin (here)             ← you register this on Claude
        │     skills/backup-restore/SKILL.md
        │
        └── OpenShift Lightspeed
              └── OpenShift MCP server
                    └── inherits OADP CLI awareness from this plugin
                          (same commands / workflows as this skill)
```

**Source of truth for CLI awareness:** `skills/backup-restore/SKILL.md`

Lightspeed/MCP maintainers should mirror this skill when adding OADP CLI tools —
not duplicate logic in the binary.

## Layout

```
migtools/oadp-cli/
├── .claude-plugin/
│   └── marketplace.json     # Marketplace registry (repo root)
└── plugins/
    └── oadp-cli/
        ├── .claude-plugin/
        │   └── plugin.json  # Plugin manifest
        ├── skills/
        │   └── backup-restore/
        │       └── SKILL.md
        └── README.md
```

## Install from marketplace

**After merge to `migtools/oadp-cli` on GitHub:**

```bash
claude plugin marketplace add github:migtools/oadp-cli
claude plugin install oadp-cli@oadp-cli-plugins
```

Restart Claude Code, then invoke `/oadp-cli:backup-restore` or ask about OADP backup/restore.

**Local dev (before merge):**

```bash
claude plugin marketplace add ~/git/oadp-cli
claude plugin install oadp-cli@oadp-cli-plugins
```

**One-session test (no marketplace):**

```bash
claude --plugin-dir ~/git/oadp-cli/plugins/oadp-cli
```

## Register on enterprise Claude

Provide to the platform team:

- **Marketplace:** `github:migtools/oadp-cli`
- **Marketplace name:** `oadp-cli-plugins`
- **Plugin:** `oadp-cli@oadp-cli-plugins`
- **Manifest:** `.claude-plugin/marketplace.json` at repo root

## Relationship to Lightspeed MCP (follow-on)

OpenShift Lightspeed registers the OpenShift MCP server in `OLSConfig` (see
[Red Hat OLS configure docs](https://docs.redhat.com/en/documentation/red_hat_openshift_lightspeed/1.0/html-single/configure/index)).

That MCP layer should inherit the **OADP CLI awareness** documented here — the same
install, `setup`, and command patterns documented in `skills/backup-restore/SKILL.md`.

This plugin lands first; MCP/Lightspeed reuses the content rather than
embedding AI in `oadp-cli`.

## Maintenance

- Update `skills/backup-restore/SKILL.md` when CLI commands or docs change.
- Keep MCP/Lightspeed tool definitions aligned with the skill.
- Bump `version` in `.claude-plugin/plugin.json` for marketplace updates.

## References

- [Claude Code plugins](https://code.claude.com/docs/en/plugins)
- [OADP CLI docs](https://github.com/migtools/oadp-cli/tree/main/docs)
