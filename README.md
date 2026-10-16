# ansi-navi

**A keyboard-driven launcher for Ansible playbooks.**

Stop memorising playbook paths and flag combinations. ansi-navi gives you a
fuzzy search palette over every playbook in your project, opens a guided
parameter wizard with live command preview, and runs ansible-playbook — all
without leaving your terminal.

Ships as a single static binary. No runtime, no dependencies, no install step
beyond dropping the file on your PATH.

---

## Features

- **Fuzzy search** — type any fragment of a playbook name and results rank
  instantly with matched characters highlighted.
- **Run wizard** — every parameter on one focused screen: text fields, enum
  dropdowns, boolean toggles. Required fields are flagged; the command is
  previewed live as you type.
- **Host picker** — checkbox multi-select over your inventory hosts and
  groups, loaded straight from your inventory file.
- **Live output** — ansible-playbook runs in the foreground with full output
  visible; the TUI waits for you to press Enter before repainting so nothing
  scrolls away.
- **Shell escape hatch** — `Ctrl+T` drops into an interactive shell and carries
  environment variables (e.g. `$VAULT_PASSWORD_FILE`) back into the session
  when you return.
- **One binary, everywhere** — statically compiled; works on Windows, macOS,
  and Linux with identical behaviour.

---

## Installation

Download the binary for your platform from the
[releases page](https://github.com/phr34k/ansi-navi/releases) and place it
anywhere on your `PATH`.

Or build from source (requires Go 1.21+):

```sh
go install github.com/phr34k/ansi-navi@latest
```

`ansible-playbook` must be on your `PATH`.

---

## Usage

```sh
# Run from inside an Ansible project directory
ansi-navi

# Point at a project from anywhere
ansi-navi --project /etc/ansible
ansi-navi --project C:\ops\ansible

# Pre-populate the limit field
ansi-navi --limit web1,web2
ansi-navi --limit all
```

| Key | Action |
|---|---|
| Type | Filter playbooks |
| `↑` / `↓` | Move selection |
| `Enter` | Open run wizard |
| `Ctrl+R` | Run immediately (wizard) |
| `Space` | Open host picker (limit field) |
| `Ctrl+T` | Drop into shell |
| `Esc` | Clear search / back / quit |

---

## Building from source

```sh
git clone https://github.com/phr34k/ansi-navi
cd ansi-navi
go build -o ansi-navi .
```

Cross-compile (fully static, no CGO):

```sh
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ansi-navi.exe .
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o ansi-navi     .
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o ansi-navi-mac .
```

---

## License

Apache License 2.0 with Additional Conditions — see [LICENSE](LICENSE).  
Redistribution must preserve the donation mechanism unchanged.

---

## Support

ansi-navi is free and open-source. If it saves you time, please consider
[supporting the project](https://github.com/sponsors/phr34k). ♥
