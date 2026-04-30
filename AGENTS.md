# AGENTS.md

## Planning Workflow

Before starting any significant feature or requirement modification:

1. **Check Existing Requirement Doc**
   - First check `docs/requirements/` for an existing matching document
   - If found, read it directly to understand the requirement (avoid re-searching codebase)
   - Only if no document exists, perform codebase search, then immediately create the document

2. **Create Planning Document**
   - Create `plans/{feature-name}-{date}.md` (date format: YYYYMMDD) with:
     - Overview of the change
     - Affected components and files
     - Implementation steps
     - Testing strategy
     - Expected deliverables
     - Link to requirement doc: `docs/requirements/{slug}.md`

3. **Review with User**
   - Present the plan and get approval before execution

4. **Execute According to Plan**
   - Follow the implementation steps
   - Update plan if adjustments are needed

5. **Update Documentation After Completion**
   - Update `docs/CHANGELOG.md` with:
     - Feature name and date
     - Link to plan file (`plans/xxx.md`)
     - Link to requirement doc (`docs/requirements/xxx.md`)
     - Summary of changes made
     - Affected components
   - Update `docs/ROADMAP.md` checklist if applicable (mark `[x]`)
   - Update `docs/requirements/{slug}.md` with change history

## Requirement Doc Convention

All requirement documents are stored in `docs/requirements/{slug}.md` with a fixed template:

```markdown
# {Requirement Name}

## Original Requirement
[What was requested]

## Scope
### In Scope
- ...

### Out of Scope
- ...

## Key Decisions & Tradeoffs
- ...

## Affected Components
- ...

## Change History
| Date | Change | Reason |
|------|--------|--------|
| YYYY-MM-DD | Initial creation | Original requirement |

## Related Links
- Plan: `plans/{feature-name}-{date}.md`
- Related: ...
```

When modifying an existing requirement, ALWAYS read `docs/requirements/{slug}.md` first.

## Notable Conventions

- Uses Air for backend hot reload (configured in `.air.toml`)
- Go 1.24.11 with CGO required
- Docreader runs as separate gRPC service on port 50051
- Uses `golang-migrate` for database migrations
- Skills run in Docker sandbox by default for security isolation