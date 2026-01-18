# Ralph Agent Instructions

## Your Task

1. Read `ralph/prd.json`
2. Read `ralph/progress.txt` (check Codebase Patterns first)
3. **Read `AGENTS.md`** to understand architecture and constraints
4. Verify you're on the correct branch
5. Pick highest priority story where `status: "incomplete"`
6. **Update prd.json: `status: "in-progress"`**
7. Implement that ONE story completely:
   - Write valid, type-safe Go code, following best practices
   - Add tests for new functionality
   - Follow patterns from AGENTS.md and progress.txt
8. **Verify implementation:**
   - Run `go build` (must pass)
   - Run `go test` (must pass)
   - Fix any errors before proceeding
9. Update relevant AGENTS.md files with new learnings/patterns
10. Commit: `feat: [ID] - [Title]`
11. **Update prd.json: `status: "complete"`**
12. Append learnings to progress.txt

## Progress Format

APPEND to progress.txt:

```
## [Date] - [Story ID]
- **What was implemented:** [description]
- **Files changed:** [list]
- **Tests added:** [test files/descriptions]
- **Learnings:**
  - Patterns discovered
  - Gotchas encountered
  - AGENTS.md updates made
---
```

## Codebase Patterns

Maintain at the TOP of progress.txt:

## Quality Gates

Before marking `status: "complete"`:

- ✅ Go compiles without errors
- ✅ All tests pass
- ✅ Code follows AGENTS.md conventions
- ✅ New patterns documented in AGENTS.md
- ✅ Learnings captured in progress.txt

## Stop Condition

If ALL stories have `status: "complete"`, reply:
<promise>COMPLETE</promise>

Otherwise end normally after completing ONE story.
