---
name: generate-commit
description: Generate a Conventional Commits-compliant commit message by analyzing already-staged git changes. Does not make any code changes, does not run any git write/update commands, and outputs only the commit message. Use when the user asks for a commit message, says "generate commit", "write commit", "commit message for staged changes", or similar.
disable-model-invocation: true
---

# Generate Commit Message

Generate a well-formed commit message for already-staged changes following the [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/) specification.

## Rules

- **No code changes.** Do not modify any file.
- **No git write commands.** Do not run `git commit`, `git add`, `git reset`, `git stash`, or any other mutating git command.
- **Output only the commit message** — no preamble, no explanation, no markdown fences around it.

## Workflow

1. Run `git diff --cached` to read the staged diff.
2. Run `git status --short` to see which files are staged.
3. Analyse the diff to determine the dominant change type and scope.
4. Compose the commit message using the format below.
5. Output the message and stop.

## Commit Message Format

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Type

| Type | When to use |
|------|-------------|
| `feat` | New feature or capability |
| `fix` | Bug fix |
| `refactor` | Code restructuring without behaviour change |
| `test` | Adding or updating tests |
| `docs` | Documentation only |
| `chore` | Build, tooling, dependency updates |
| `perf` | Performance improvement |
| `ci` | CI/CD configuration |
| `style` | Formatting, whitespace (no logic change) |
| `revert` | Reverting a previous commit |

### Scope

Derive from the primary package, module, or directory affected (e.g. `auth`, `product`, `storage`, `api`). Omit if changes span the entire codebase.

### Description

- Imperative mood, present tense: "add", "fix", "update" — not "added", "fixes"
- No period at the end
- Max ~72 characters

### Body (optional)

Include when the *why* or *what* is non-obvious. Wrap at 72 characters.

### Footer (optional)

- Reference issues: `Refs: #123`
- Breaking changes: `BREAKING CHANGE: <description>` or append `!` after type/scope

## Examples

**Simple feature:**
```
feat(product): add product-media association endpoint
```

**Bug fix with body:**
```
fix(storage): handle nil pointer when provider config is missing

Storage config loader assumed a non-nil default provider was always
present. Added a guard and a meaningful error message.
```

**Breaking change:**
```
feat(auth)!: replace session tokens with JWT

BREAKING CHANGE: clients must send Authorization: Bearer <token>
instead of X-Session-Token header.
```

**Chore:**
```
chore(deps): upgrade GORM to v2.0.10
```

## Output

Return only the raw commit message text — no surrounding explanation, no code fences.
