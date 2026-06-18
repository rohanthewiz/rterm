# Session: Working around the Xcode license prompt blocking git (no admin)

**Date:** 2026-06-15 11:54 · **Session ID:** `68ab43e2-6b51-4377-83f3-bbf83e9e71a9`

## Problem

On a work laptop **without admin access**, the user installed Xcode via the App Store. macOS now demands acknowledgment of the Xcode license, which requires admin. As a side effect, `git` became blocked with:

```
You have not agreed to the Xcode license agreements. Please run
'sudo xcodebuild -license' from within a Terminal window to review
and agree to the Xcode and Apple SDKs license.
```

## Root cause

`/usr/bin/git` is an Apple **shim**. It routes through the **active developer directory**, which `xcode-select` had pointed at `/Applications/Xcode.app/Contents/Developer`. Xcode requires license acceptance (admin), so every git invocation was gated.

## Diagnosis (commands run)

```sh
which -a git              # → /usr/bin/git
xcode-select -p           # → /Applications/Xcode.app/Contents/Developer
ls -d /Library/Developer/CommandLineTools          # present
ls -l /Library/Developer/CommandLineTools/usr/bin/git   # present (Apple Git-155)
echo "$DEVELOPER_DIR"     # unset
```

Key finding — the standalone **Command Line Tools** are installed at
`/Library/Developer/CommandLineTools` and do **not** require license acceptance:

```sh
DEVELOPER_DIR=/Library/Developer/CommandLineTools git --version
# → git version 2.50.1 (Apple Git-155)   ✅  (no license prompt)

git --version
# → blocked with the Xcode license error  ❌
```

## Why this workaround instead of others

- `xcode-select --switch ...` would fix it globally **but needs admin** — not available.
- The **`DEVELOPER_DIR` environment variable** overrides `xcode-select` for the **user only**, no admin required. It redirects `git`, `clang`, `xcrun`, etc. to the Command Line Tools.
- Do **not** delete Xcode to "fix" it — unnecessary; the CLT path is independent of Xcode.
- Trade-off: Xcode-specific SDKs (iOS simulators, etc.) are not active in the shell. Irrelevant for Go/git/general CLI work — CLT ships the compilers and macOS SDK needed.
- Accepting the Xcode license itself genuinely needs admin (`sudo xcodebuild -license`) — would require IT, but isn't needed here.

## Fix applied

Appended to `~/.zshrc`:

```sh
# Use Command Line Tools instead of Xcode (avoids Xcode license prompt; no admin needed)
export DEVELOPER_DIR=/Library/Developer/CommandLineTools
```

Confirmed it wasn't already present before appending.

## Follow-ups for the user

- Applies automatically to every **new** terminal. For the current shell: `source ~/.zshrc`.
- The running Claude Code session won't pick up the new env until restarted; in the meantime git commands can be prefixed with `DEVELOPER_DIR=/Library/Developer/CommandLineTools ...`.
