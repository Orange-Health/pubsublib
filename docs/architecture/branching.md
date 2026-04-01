# Branching Strategy

---

## Branch Model

```mermaid
gitGraph
   commit id: "init"

   branch dev
   checkout dev
   commit id: "dev baseline"

   branch feature/OH-1001-redis-adapter
   checkout feature/OH-1001-redis-adapter
   commit id: "add redis adapter"
   commit id: "tests"
   checkout dev
   merge feature/OH-1001-redis-adapter id: "merge feature → dev"

   branch feature/OH-1002-compression
   checkout feature/OH-1002-compression
   commit id: "gzip compress"
   commit id: "base64 encode"
   checkout dev
   merge feature/OH-1002-compression id: "merge compression → dev"

   branch release
   checkout release
   merge dev id: "cut release from dev"
   commit id: "RC-26.0401.001"

   checkout main
   merge release id: "promote to main" tag: "26.0401.1"

   branch hotfix/OH-5001-overflow-ttl
   checkout hotfix/OH-5001-overflow-ttl
   commit id: "fix redis TTL"
   checkout main
   merge hotfix/OH-5001-overflow-ttl id: "hotfix → main" tag: "26.0401.2"
   checkout dev
   merge hotfix/OH-5001-overflow-ttl id: "hotfix → dev"
   checkout release
   merge hotfix/OH-5001-overflow-ttl id: "hotfix → release"
```

---

## Branch Definitions

| Branch | Created from | Merges into | Purpose |
|---|---|---|---|
| `main` | — | — | Production-grade code; every commit is a releasable version |
| `dev` | `main` | `release` | Integration branch; all feature work lands here first |
| `release` | `dev` | `main` | Release stabilisation; only bug fixes committed here |
| `feature/<JIRAID>-<name>` | `dev` | `dev` | New features or enhancements |
| `hotfix/<JIRAID>-<desc>` | `main` | `main` + `dev` + `release` | Critical production bug fixes |

---

## Branch Naming Conventions

### Feature branches
```
feature/<JIRA-ID>-<kebab-case-description>
```
Examples:
- `feature/OH-1234-redis-overflow`
- `feature/SB-345-gzip-compression`

### Hotfix branches
```
hotfix/<JIRA-ID>-<kebab-case-description>
```
Examples:
- `hotfix/OH-5001-fix-message-ttl`
- `hotfix/OH-5002-md5-mismatch`

---

## Versioning Scheme

### Production releases (semver-compatible Go module tags)
```
YY.MMDD.PATCHCOUNT
```

| Segment | Description | Example |
|---|---|---|
| `YY` | Two-digit year | `26` (2026) |
| `MMDD` | Zero-padded month + day | `0401` (April 1st) |
| `PATCHCOUNT` | Sequential patch number starting at 1 | `1`, `2`, `3` |

Full examples: `v26.0401.1`, `v26.0401.2`

> The existing `v0.3.x` semver tags (e.g. `v0.3.3`) represent the legacy versioning scheme prior to this convention.

### Release candidates
```
RC-YY.MMDD.BUILDCOUNT
```

Examples: `RC-26.0401.001`, `RC-26.0401.002`

Used on the `release` branch during stabilisation.

### Staging / sandbox builds
```
vstag-b<NN>          # staging
vstagsandbox-b<NN>   # sandbox
stag-<desc>          # descriptive staging tag
```

Examples: `vstag-b14`, `vstagsandbox-b03`, `stag-compression-test`

---

## Lifecycle Rules

1. **Feature branches** are short-lived — branch from `dev`, open a PR back to `dev`, delete after merge.
2. **Release branches** are cut when `dev` is stable; only bug fixes are committed directly; promoted to `main` via PR when ready.
3. **Hotfixes** must be cherry-picked (or merged) into **all** three long-running branches: `main`, `dev`, and `release`.
4. **`main` is protected** — direct pushes are not allowed; all changes arrive via PR.
5. **Tags are immutable** — never delete or move an existing version tag.
