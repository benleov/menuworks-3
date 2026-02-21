# Agent Release Workflow

This document defines the **automated release process** managed by the agent (GitHub Copilot) for MenuWorks. This workflow eliminates manual version management and ensures consistent, repeatable releases.

## Overview

The agent manages the complete release lifecycle:
1. Feature development on feature branches
2. Automated testing and validation
3. User-driven manual testing
4. Version determination from conventional commits
5. Squash-merge to main
6. Git tag creation and push
7. GitHub Actions automation (build, release, VERSION sync)

## Workflow Steps

### 1. Feature Development

**Agent creates feature branch:**
```bash
git checkout main
git pull
git checkout -b feature/<feature-name>
```

**Agent implements with conventional commits:**
- `feat:` for new features → triggers MINOR bump
- `fix:` for bug fixes → triggers PATCH bump
- `docs:` for documentation changes → no version bump
- `refactor:` for code restructuring → no version bump
- `chore:` for maintenance tasks → no version bump
- `BREAKING CHANGE:` in commit body → triggers MAJOR bump

**Example commits:**
```bash
git commit -m "feat: add mouse support for menu navigation"
git commit -m "fix: prevent crash when config.yaml is empty"
git commit -m "feat!: change config format to TOML

BREAKING CHANGE: config.yaml replaced with config.toml"
```

### 2. Automated Testing

**Agent runs tests:**
```powershell
.\test.ps1
```

**Expected output:**
```
ok      github.com/benworks/menuworks/config
ok      github.com/benworks/menuworks/menu
```

**If tests fail:** Agent fixes issues and re-runs tests until all pass.

### 3. Build Verification

**Agent builds binary:**
```powershell
.\build.ps1 -Target windows -Version (Get-Content VERSION)
```

**Expected output:**
```
Building menuworks for windows (amd64)...
Build complete: dist/menuworks-windows.exe
```

### 4. Manual Testing Request

**Agent provides testing instructions to user:**
```
### Manual Testing Required

Please test the following:

1. Navigate to: cd .\dist
2. Run: .\menuworks-windows.exe
3. Test scenarios:
   - [Feature-specific test 1]
   - [Feature-specific test 2]
   - [Feature-specific test 3]
4. Verify normal operation:
   - Execute a command
   - Press R to reload config
   - Exit with Back → Quit

Reply "Approved" when ready to release.
```

### 5. Version Determination

**Agent analyzes all commits since last tag:**

```bash
# Get commits since last tag
git log $(git describe --tags --abbrev=0)..HEAD --oneline

# Determine version bump:
# - Any feat: commit → MINOR
# - Any BREAKING CHANGE → MAJOR (overrides MINOR)
# - Only fix: commits → PATCH
# - Only docs:/refactor:/chore: → no release needed
```

**Version bump logic:**
- Current version: `3.1.0`
- Contains `feat:` → `3.2.0` (MINOR)
- Contains `fix:` only → `3.1.1` (PATCH)
- Contains `BREAKING CHANGE:` → `4.0.0` (MAJOR)

### 6. Squash-Merge to Main

**Agent performs squash-merge:**
```bash
git checkout main
git pull
git merge --squash feature/<feature-name>
git commit -m "feat: <feature summary>

<detailed description>

Closes #<issue-number>"
git push origin main
```

**Result:** All feature branch commits become one commit on main with clear conventional commit message.

### 7. Create and Push Tag

**Agent creates annotated tag:**
```bash
VERSION="3.2.0"
git tag -a v$VERSION -m "release: version $VERSION"
git push origin v$VERSION
```

**This triggers GitHub Actions release workflow.**

### 8. GitHub Actions Automation

**Triggered by tag push (`v*`):**

1. **Run CI tests** (config, menu packages)
2. **Build binaries:**
   - Windows 64-bit
   - Linux 64-bit
   - macOS Intel 64-bit
   - macOS ARM64
3. **Generate checksums** (SHA256)
4. **Create GitHub Release:**
   - Auto-generated release notes from commits
   - Upload all binaries and checksums
   - Publish as latest release
5. **Sync VERSION file and README badge:**
   - Update VERSION file with new version
   - Update README badge
   - Commit back to main

### 9. Cleanup

**Agent deletes feature branch:**
```bash
git branch -d feature/<feature-name>
git push origin --delete feature/<feature-name>
```

### 10. Verification

**Agent verifies release:**
```bash
# Check VERSION file updated
cat VERSION
# Expected: 3.2.0

# Check README badge updated
grep "badge/version" README.md
# Expected: version-3.2.0-blue

# Verify release published
# Visit: https://github.com/benleov/menuworks-3/releases
```

**Agent reports to user:**
```
✓ Release v3.2.0 published successfully
✓ VERSION file synced to 3.2.0
✓ README badge updated to 3.2.0
✓ Feature branch deleted
✓ Release available at: https://github.com/benleov/menuworks-3/releases/tag/v3.2.0
```

## Conventional Commit Reference

### Commit Types

| Type | Bump | Description | Example |
|------|------|-------------|---------|
| `feat:` | MINOR | New feature | `feat: add theme support` |
| `fix:` | PATCH | Bug fix | `fix: resolve menu crash` |
| `docs:` | - | Documentation only | `docs: update README` |
| `refactor:` | - | Code restructure | `refactor: simplify navigator` |
| `chore:` | - | Maintenance | `chore: update dependencies` |
| `BREAKING CHANGE:` | MAJOR | Breaking change | (in commit body) |

### Breaking Changes

Two ways to indicate breaking changes:

**1. Exclamation mark:**
```bash
git commit -m "feat!: change config format to TOML"
```

**2. Footer:**
```bash
git commit -m "feat: change config format

BREAKING CHANGE: config.yaml replaced with config.toml"
```

## Error Handling

### Tests Fail After Merge

**Problem:** Tests passed locally but fail in CI after merge.

**Recovery:**
1. Agent creates hotfix branch: `git checkout -b hotfix/<issue>`
2. Agent fixes issue with `fix:` commit
3. Agent repeats full workflow (test → build → merge → tag)
4. New PATCH version released

### Tag Already Exists

**Problem:** Agent tries to create tag `v3.2.0` but it already exists.

**Recovery:**
1. Check if tag points to correct commit: `git show v3.2.0`
2. If incorrect, delete and recreate:
   ```bash
   git tag -d v3.2.0
   git push origin :v3.2.0
   git tag -a v3.2.0 -m "release: version 3.2.0"
   git push origin v3.2.0
   ```

### VERSION File Out of Sync

**Problem:** VERSION file shows `3.1.0` but latest tag is `v3.2.0`.

**Recovery:**
- Expected during development (VERSION syncs after tag push)
- CI logs warning but doesn't fail
- Release workflow auto-corrects on next release
- Manual fix if needed: 
  ```bash
  echo "3.2.0" > VERSION
  git commit -am "chore: sync VERSION to 3.2.0"
  git push origin main
  ```

## Agent Decision Tree

```
User approves feature
  ↓
Agent analyzes commits
  ↓
Contains feat: ? ────→ YES ──→ MINOR bump
  ↓ NO
Contains fix: ? ────→ YES ──→ PATCH bump
  ↓ NO
Contains docs:/refactor:/chore: only?
  ↓ YES
No release needed (skip to cleanup)
  ↓
Contains BREAKING CHANGE: ?
  ↓ YES
Override with MAJOR bump
  ↓
Squash-merge to main
  ↓
Create tag v<VERSION>
  ↓
Push tag
  ↓
GitHub Actions handles rest
  ↓
Clean up branches
  ↓
Report success to user
```

## User Interaction Points

The user is **only involved** at these points:

1. **Feature approval:** Confirm implementation plan aligns with intent
2. **Manual testing:** Test binary and approve functionality
3. **Release confirmation:** Agent reports release success

**Everything else is automated.**

## Benefits

✓ **No manual version editing** — Agent determines bumps from commits  
✓ **No CHANGELOG maintenance** — GitHub auto-generates release notes  
✓ **Consistent process** — Same workflow every time  
✓ **Reduced errors** — No forgotten steps or version mismatches  
✓ **Clear history** — Conventional commits enable automatic versioning  
✓ **Fast releases** — Minutes instead of manual checklist execution  

## Manual Override (Emergency)

If agent is unavailable, releases can still be triggered manually:

```bash
# Determine version manually
CURRENT=$(git describe --tags --abbrev=0)
echo "Current: $CURRENT"

# Create tag manually
git tag -a v3.2.0 -m "release: version 3.2.0"
git push origin v3.2.0

# GitHub Actions handles the rest
```

**Note:** Manual releases still follow the same automated build/release/sync process via GitHub Actions.
