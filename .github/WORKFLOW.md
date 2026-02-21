# Agent Workflow

Step-by-step instructions for the agent to develop, test, and release features in MenuWorks.

## Prerequisites

- **GitHub CLI (`gh`)** must be installed and authenticated.
  - Install: Windows `winget install --id GitHub.cli` | macOS `brew install gh` | Linux: [instructions](https://github.com/cli/cli/blob/trunk/docs/install_linux.md)
  - After installing, you may need to **restart your terminal**.
  - Authenticate: `gh auth login` (follow prompts).

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

### 1. Verify repo state

```powershell
git status --porcelain
git checkout main
git pull origin main
git fetch --tags
gh auth status
```

**Success:** `git status --porcelain` output is empty. Checkout, pull, and fetch complete without errors. `gh auth status` shows a logged-in account.

**STOP** if working copy is dirty. Tell the user what files are dirty.

**STOP** if `gh` is not recognised ("not recognized as a name of a cmdlet" / "command not found"). Tell the user to install `gh` (see Prerequisites) and restart their terminal.

**STOP** if `gh` reports "To get started with GitHub CLI, please run: gh auth login". Tell the user to run `gh auth login`.

### 2. Determine version

Read the current version:

```powershell
git describe --tags --abbrev=0
```

**STOP** if this command fails (e.g. no tags exist). Fall back to reading the VERSION file (`Get-Content VERSION`). If both fail, ask the user for the current version.

The output includes a `v` prefix (e.g. `v3.1.0`). Strip it to get the bare version number (e.g. `3.1.0`).

Ask the user: **What type of change is this?**
- `feat` → MINOR bump (e.g. 3.1.0 → 3.2.0)
- `fix` → PATCH bump (e.g. 3.1.0 → 3.1.1)
- `breaking` → MAJOR bump (e.g. 3.1.0 → 4.0.0)
- `docs` / `refactor` / `chore` → no version bump

If `docs`/`refactor`/`chore` only: inform the user no release is needed. The workflow may still proceed (branch, implement, PR) but skip version bumping and tagging in Step 10. Ask the user whether to continue or stop.

Calculate the new version number and confirm it with the user before continuing.

### 3. Create feature branch

Derive the branch name from the feature description (lowercase, hyphens). Confirm with the user if the name is unclear.

```powershell
git checkout -b feature/<feature-name>
```

**STOP** if branch creation fails (e.g. branch already exists). Inform the user and ask how to proceed.

### 4. Create implementation plan

Research the codebase, then present a plan to the user containing:
1. Files to modify (with specific functions/areas)
2. Changes to make (concise descriptions)
3. Test scenarios (automated - adding new unit tests to cover new functionality, or updating existing + manual tests for the user to verify)

**STOP** until the user approves the plan.

### 5. Implement

Make changes and commit with conventional commits. One or more commits on the feature branch are fine; they will be squashed when the PR is merged.

```powershell
git add <files>
git commit -m "<type>: <description>"
```

Do **NOT** update the VERSION file.

### 6. Run tests

**Skip this step** if no `.go` files were changed (`git diff main --name-only | Select-String '\.go$'` returns nothing). Docs-only changes do not need testing.

```powershell
.\test.ps1
```

**Success:** All packages show `ok` status, exit code 0.
**STOP** if any test fails. Fix and re-run before continuing.

### 7. Build binary

**Skip this step** if no `.go` files were changed.

```powershell
.\build.ps1 -Target windows -Version <new-version>
```

Use the version determined in Step 2. For no-release changes, use the current version.

**Success:** `dist/menuworks-windows.exe` exists and build output shows no errors.
**STOP** if build fails. Fix and re-run.

### 8. Request manual testing

**Skip this step** if no `.go` files were changed.

Provide the user with exact testing instructions:

```
1. cd .\dist
2. .\menuworks-windows.exe
3. Test the feature:
   - [Feature-specific scenario 1]
   - [Feature-specific scenario 2]
4. Regression test:
   - Navigate between menus and submenus
   - Execute a command and verify output
   - Press R to reload config
   - Exit via Back/Quit
```

**STOP** until the user confirms the feature works correctly.

### 9. Push and create PR

If any changes were made after manual testing (e.g. fixes from user feedback), commit them before pushing.

```powershell
git push origin feature/<feature-name>
```

Create the PR using `gh`, with the body following the project's PR template (`.github/PULL_REQUEST_TEMPLATE.md`):

```powershell
gh pr create --base main --title "<type>: <description>" --body @'
## Description
<concise summary of what this PR changes>

## Type of Change
- [x] <check the applicable type: Bug fix / New feature / Enhancement / Documentation update>

## Testing
- [x] All tests pass (or "Skipped — docs-only change")
- [x] Builds successfully (or "Skipped — docs-only change")
- [x] Manually tested (or "Skipped — docs-only change")
- [x] Documentation updated (if needed)

## Related Changes
<list files/areas affected>
'@
```

**STOP** if PR creation fails. Inform the user.

Wait for CI checks to pass (do not use `--watch` as it uses an alternate screen buffer that prevents output capture):

```powershell
gh pr checks
```

If checks are still pending, wait and re-run `gh pr checks` until they complete.

**STOP** if CI checks fail. Inform the user of the failure details.

Report the PR URL to the user. Inform them: **merging this PR will trigger the release pipeline** (or "no release" for docs-only changes).

### 10. Merge and release

Ask the user for approval to merge the PR.

**STOP** until the user approves.

Merge via `gh` (squash merge, auto-deletes remote branch):

```powershell
gh pr merge --squash --delete-branch
```

Sync local repo:

```powershell
git checkout main
git pull origin main
```

Tag the release (skip for `docs`/`refactor`/`chore` changes):

```powershell
git tag -a v<VERSION> -m "release: version <VERSION>"
git push origin v<VERSION>
```

GitHub Actions will automatically:
- Build binaries (Windows, Linux, macOS Intel, macOS ARM)
- Generate SHA256 checksums
- Create GitHub Release with auto-generated notes
- Sync VERSION file back to main

Clean up local branch:

```powershell
# Use -D because squash-merge means git won't recognise the branch as merged
git branch -D feature/<feature-name>
```

Ask the user to verify the release is published at:

```
https://github.com/benleov/menuworks-3/releases
```

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

**`gh` not installed:**
User sees "not recognized" / "command not found". Install `gh` (see Prerequisites), restart terminal.

**`gh` not authenticated:**
User sees "please run: gh auth login". Run `gh auth login` and follow prompts.

**Tests fail after merge to main:**
Create a hotfix branch (`hotfix/<issue>`), fix with a `fix:` commit, repeat workflow from Step 6.

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
