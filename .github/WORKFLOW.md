# Agent Workflow

Step-by-step instructions for the agent to develop, test, and release features in MenuWorks.

## Rules

- Never push directly to `main`. Always use a feature branch.
- Never edit the VERSION file manually. Git tags are the authoritative version source.
- Use conventional commits on all feature branch commits.
- If any step fails or is ambiguous, **STOP** and inform the user what happened before proceeding.
- Go is installed at `bin\go\bin\go.exe`, not in PATH.
- Build: `.\build.ps1 -Target windows -Version <version>`
- Test: `.\test.ps1`

---

## Steps

### 1. Verify clean working copy

```powershell
git status --porcelain
```

**Success:** Output is empty.
**STOP** if output is non-empty. Tell the user what files are dirty.

### 2. Switch to main

```powershell
git checkout main
```

### 3. Pull latest

```powershell
git pull origin main
```

### 4. Determine version

Read the current version:

```powershell
git describe --tags --abbrev=0
```

Ask the user: **What type of change is this?**
- `feat` → MINOR bump (e.g. 3.1.0 → 3.2.0)
- `fix` → PATCH bump (e.g. 3.1.0 → 3.1.1)
- `breaking` → MAJOR bump (e.g. 3.1.0 → 4.0.0)
- `docs` / `refactor` / `chore` → no version bump

**STOP** if docs/refactor/chore only. Inform the user no release is needed and ask how to proceed.

Calculate the new version number and confirm it with the user before continuing.

### 5. Create feature branch

```powershell
git checkout -b feature/<feature-name>
```

Branch name should be lowercase with hyphens (e.g. `feature/theme-support`).

### 6. Create implementation plan

Research the codebase, then present a plan to the user containing:
1. Files to modify (with specific functions/areas)
2. Changes to make (concise descriptions)
3. Test scenarios (automated + manual)

**STOP** until the user approves the plan.

### 7. Implement

Make changes and commit with conventional commits:

```powershell
git add <files>
git commit -m "<type>: <description>"
```

Do **NOT** update the VERSION file.

### 8. Run tests

```powershell
.\test.ps1
```

**Success:** All packages show `ok` status, exit code 0.
**STOP** if any test fails. Fix and re-run before continuing.

### 9. Build binary

```powershell
.\build.ps1 -Target windows -Version <new-version>
```

Use the version determined in Step 4.

**Success:** `dist/menuworks-windows.exe` exists and build output shows no errors.
**STOP** if build fails. Fix and re-run.

### 10. Request manual testing

Provide the user with exact testing instructions:

```
1. cd .\dist
2. .\menuworks-windows.exe
3. Test:
   - [Feature-specific scenario 1]
   - [Feature-specific scenario 2]
   - [Feature-specific scenario 3]
4. Verify normal operation:
   - Execute a command
   - Press R to reload config
   - Exit via Back/Quit
```

**STOP** until the user confirms the feature works correctly.

### 11. Push and provide PR summary

Commit any remaining changes, then push:

```powershell
git push origin feature/<feature-name>
```

Present the user with a PR summary in a code fence:

````
```
Title: <conventional commit style title>
Version: <current> → <new>
Type: feat|fix|breaking

## Changes
- <change 1>
- <change 2>

## Testing
- Automated: all tests pass
- Manual: user verified on Windows
```
````

Provide the PR creation link:

```
https://github.com/benleov/menuworks-3/pull/new/feature/<feature-name>
```

Inform the user: **Merging this PR will trigger the release pipeline.**

### 12. Post-merge release

After the user merges the PR on GitHub:

```powershell
git checkout main
git pull origin main
git tag -a v<VERSION> -m "release: version <VERSION>"
git push origin v<VERSION>
```

GitHub Actions will automatically:
- Build binaries (Windows, Linux, macOS Intel, macOS ARM)
- Generate SHA256 checksums
- Create GitHub Release with auto-generated notes
- Sync VERSION file back to main

Clean up:

```powershell
git branch -d feature/<feature-name>
git push origin --delete feature/<feature-name>
```

Verify the release was published and report to the user.

---

## Conventional Commits Reference

| Type | Version Bump | Example |
|------|-------------|---------|
| `feat:` | MINOR | `feat: add theme support` |
| `fix:` | PATCH | `fix: resolve menu crash` |
| `feat!:` or `BREAKING CHANGE:` in body | MAJOR | `feat!: change config format` |
| `docs:` | none | `docs: update README` |
| `refactor:` | none | `refactor: simplify navigator` |
| `chore:` | none | `chore: update dependencies` |

---

## Error Recovery

**Tests fail after merge to main:**
Create a hotfix branch (`hotfix/<issue>`), fix with a `fix:` commit, repeat workflow from Step 8.

**Tag already exists:**
```powershell
git tag -d v<VERSION>
git push origin :v<VERSION>
```
Then recreate the tag.

**Build fails (binary locked):**
```powershell
Get-Process | Where-Object {$_.ProcessName -like "*menuworks*"} | Stop-Process -Force
```
Then retry the build.

---

## Manual Override (agent unavailable)

```powershell
git tag -a v<VERSION> -m "release: version <VERSION>"
git push origin v<VERSION>
# GitHub Actions handles the rest
```
