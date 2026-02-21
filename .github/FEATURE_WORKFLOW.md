# Feature Development & Release Workflow

This document provides a step-by-step process for developing and releasing features in MenuWorks.

## Step 1: Prepare Your Repository

Ensure your local repository is up to date and in a clean state:

```powershell
git checkout main
git pull origin main
git status
```

**Verify:** `git status` must show `nothing to commit, working tree clean`.

## Step 2: Identify Feature & Version Bump

- **Determine feature name** (lowercase, hyphens): e.g., `combined-menu-title`, `mouse-support`, `config-validation`
- **Current version location:** [VERSION](../VERSION)
- **Determine Version Bump:**
  - **MAJOR:** Breaking changes (e.g., config format change) → e.g., `3.0.0` → `4.0.0`
  - **MINOR:** New feature, backward compatible → e.g., `3.0.1` → `3.1.0`  
  - **PATCH:** Bug fix only → e.g., `3.0.0` → `3.0.1`

## Step 3: Create Feature Branch

```powershell
git checkout -b feature/<feature-name>
```

Example:
```powershell
git checkout -b feature/combined-menu-title
```

**Verify:** `git branch` shows your new branch marked with `*`.

## Step 4: Create & Confirm Implementation Plan

Before implementing, create a detailed plan that includes:

1. **Files to modify** (with specific functions/lines)
2. **Changes to make** (concise descriptions)
3. **Tests to run** (manual + automated)
4. **Version bump** (MAJOR/MINOR/PATCH)
5. **CHANGELOG entry** (brief description of changes)

**Ask user for confirmation:**
> Does this plan align with your intent?

**Only proceed if user confirms.** If they request changes, update the plan.

---

## Step 5: Implement the Feature

Execute the implementation plan step-by-step:

1. Modify required source files
2. Update VERSION file to new version number
3. Add CHANGELOG.md entry under `[Unreleased]` → new `[X.Y.Z] - YYYY-MM-DD` section
4. Make git commits with clear messages following conventional commit style:
   - `feat:` for new features
   - `fix:` for bug fixes  
   - `refactor:` for code restructuring
   - `docs:` for documentation changes

---

## Step 6: Automated Testing

Run the full test suite:

```powershell
.\test.ps1
```

**Verify:** All tests pass (`ok` status, exit code 0). If tests fail, fix the issues before proceeding.

---

## Step 7: Build the Binary

```powershell
.\build.ps1 -Target windows -Version (Get-Content VERSION)
```

**Verify:** Build completes successfully. Output file at `dist/menuworks-windows.exe` should exist and be recent.

---

## Step 8: Manual Testing Instructions

Provide the user with exact commands to manually test the feature:

```
### Testing Instructions

1. **Navigate to dist folder:**
   cd .\dist

2. **Run the binary:**
   .\menuworks-windows.exe

3. **Test the following scenarios:**
   - [Specific test 1: e.g., "Navigate to Applications submenu and verify title displays 'MenuWorks 3.X - Applications'"]
   - [Specific test 2]
   - [Specific test 3]

4. **Return to main menu:**
   Press Left arrow or Esc

5. **Verify normal operation:**
   - Execute a command and confirm output displays correctly
   - Press R to reload config and verify changes persist
   - Exit with Back → Quit

6. **Confirm the feature works as expected before merging.**
```

---

## Step 9: User Manual Testing & Approval

**Wait for user to:**
1. Run the binary
2. Perform manual tests from Step 8
3. Confirm feature works correctly
4. Give approval to proceed: "Ready to merge" or "Ready for PR"

---

## Step 10: Prepare PR Summary

Create a clear PR summary with:

```markdown
## Title
[Feature Name]: [Brief description]

## Description
Explain what the feature does and why it was added.

## Changes
- [Specific change 1]
- [Specific change 2]
- Version bumped from X.Y.Z to A.B.C

## Testing
- [Automated tests: .\test.ps1 passes]
- [Manual tests: Feature verified as working]

## Files Modified
- [file1.go](file1.go) - [what changed]
- [file2.go](file2.go) - [what changed]
- VERSION - bumped to A.B.C
- CHANGELOG.md - added [A.B.C] entry
```

Provide this summary to the user and provide the GitHub PR creation link.

---

## Step 11: Create Pull Request

On GitHub:

1. Go to: https://github.com/benleov/menuworks-3/pull/new/feature/<feature-name>
2. Click **Create pull request**
3. Fill in:
   - **Title:** Use the title from Step 10
   - **Description:** Use the description from Step 10
4. Click **Create pull request**

---

## Step 12: Merge & Clean Up

**After PR approval on GitHub:**

```powershell
# Switch to main
git checkout main

# Pull latest (should include merged PR)
git pull origin main

# Verify merge was successful
git log --oneline -5

# Delete local feature branch
git branch -d feature/<feature-name>

# Delete remote feature branch
git push origin --delete feature/<feature-name>
```

---

## Step 13: Release to GitHub

**On GitHub:**

1. Go to: https://github.com/benleov/menuworks-3/releases
2. Click **Draft a new release**
3. Select tag dropdown → choose version tag (e.g., `v3.1.0`)
   - Or type `v<VERSION>` to create new tag from main
4. Fill in:
   - **Release title:** `MenuWorks <VERSION>` (e.g., `MenuWorks 3.1.0`)
   - **Description:** Copy CHANGELOG.md entry for this version
5. Click **Publish release**

---

## Checklist: Before Proceeding to Next Step

Use this checklist throughout the workflow:

- [ ] Repository is up to date (`git pull origin main`)
- [ ] Feature branch created: `feature/<feature-name>`
- [ ] Implementation plan created and **user confirmed**
- [ ] All source changes completed
- [ ] VERSION file updated
- [ ] CHANGELOG.md updated with new entry
- [ ] All commits made with clear messages
- [ ] Automated tests pass: `.\test.ps1`
- [ ] Binary builds successfully: `.\build.ps1`
- [ ] Manual tests completed and verified
- [ ] **User approves changes**
- [ ] PR created with summary
- [ ] PR merged with squash
- [ ] Local & remote feature branch deleted
- [ ] Release published on GitHub

---

## Common Issues & Recovery

### Build fails after code changes
- Check compiler errors in output
- Fix the issue in the source file
- Re-run: `.\build.ps1 -Target windows -Version (Get-Content VERSION)`

### Tests fail
- Review test output to identify which test failed
- Fix the issue in source or test file
- Re-run: `.\test.ps1`

### Binary file locked during build
```powershell
# Kill any running instance
Get-Process | Where-Object {$_.ProcessName -like "*menuworks*"} | Stop-Process -Force

# Retry build
.\build.ps1 -Target windows -Version (Get-Content VERSION)
```

### Need to undo commits before PR
```powershell
# View recent commits
git log --oneline -10

# Reset to specific commit (replace COMMIT_HASH)
git reset --soft COMMIT_HASH

# Or reset to last pushed main
git reset --soft origin/main
```

### PR merged but forgot to update something
- Create a new feature branch for the follow-up fix
- Repeat this entire workflow for the correction
- Do NOT attempt to revise a merged PR

---

## Workflow Diagram

```
START
  ↓
[1] Prepare Repo (git pull, clean status)
  ↓
[2] Identify Feature & Version Bump
  ↓
[3] Create Feature Branch
  ↓
[4] Create Implementation Plan → [User Confirms?] → NO → [Revise Plan] → back to [4]
                                    ↓ YES
[5] Implement Feature
  ↓
[6] Run Automated Tests → [Pass?] → NO → [Fix Issues] → back to [6]
                           ↓ YES
[7] Build Binary → [Success?] → NO → [Fix Build] → back to [7]
                     ↓ YES
[8] Provide Manual Testing Instructions
  ↓
[9] Wait for User Manual Testing → [Approve?] → NO → [Debug] → back to [5]
                                      ↓ YES
[10] Create PR Summary
  ↓
[11] Create Pull Request on GitHub
  ↓
[12] Merge PR (Squash) & Clean Up
  ↓
[13] Release on GitHub
  ↓
END (Version released)
```

---

## Example: Full Feature Workflow Session

**Scenario:** Add mouse support feature (MINOR version bump from 3.0.1 to 3.1.0)

```
# Step 1-3: Prepare
$ git checkout main
$ git pull origin main
$ git checkout -b feature/mouse-support

# Step 4: Create plan
[Plan created: Add mouse click support, handle resizing, UI updates, etc.]
User: "Ready to proceed with this plan"

# Step 5: Implement
[Modified: menu/navigator.go, ui/menu.go, ui/screen.go]
[Updated: VERSION to 3.1.0, CHANGELOG.md with new entry]
$ git add ...
$ git commit -m "feat: add mouse click support for menu navigation"

# Step 6: Test
$ .\test.ps1
# Output: ok github.com/benworks/menuworks/menu ... ✓
# Output: ok github.com/benworks/menuworks/config ... ✓

# Step 7: Build
$ .\build.ps1 -Target windows -Version 3.1.0
# Output: ✓ menuworks-windows.exe (3.09 MB) ✓

# Step 8-9: Manual Testing
User runs:
$ cd .\dist
$ .\menuworks-windows.exe
[Tests mouse clicks on menu items, resizing, etc.]
User: "Feature works perfectly!"

# Step 10-11: PR
[PR created with title "Mouse Support: Enable menu navigation via mouse clicks"]
[User reviews and approves on GitHub]

# Step 12-13: Release
$ git checkout main
$ git pull origin main
$ git branch -d feature/mouse-support
$ git push origin --delete feature/mouse-support
[Released on GitHub as v3.1.0]
```

