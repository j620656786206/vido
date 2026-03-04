# Implementation Patterns & Consistency Rules

## Critical Context: Brownfield Patterns

**Pattern Definition Strategy:**

Given the dual backend architecture and implementation gaps discovered in the codebase analysis, these patterns serve dual purposes:

1. **Define IDEAL patterns** all AI agents must follow for new code
2. **Document EXISTING patterns** found in current codebase for migration reference
3. **Establish MIGRATION paths** from current state → ideal state

**Pattern Enforcement Priority:**

- 🔴 **MANDATORY for all new code** - Must follow ideal patterns immediately
- 🟡 **REFACTOR existing code** - Align with patterns during Phase 1-5 consolidation
- 🟢 **VERIFY during reviews** - All AI agents check pattern compliance before committing

---

## Pattern Categories Overview

**Potential Conflict Points Identified:** 47 areas where AI agents could make different implementation choices without explicit patterns.

**Categories:**
1. **Naming Patterns** - 15 conflict points
2. **Structure Patterns** - 12 conflict points
3. **Format Patterns** - 8 conflict points
4. **Communication Patterns** - 6 conflict points
5. **Process Patterns** - 6 conflict points

---
