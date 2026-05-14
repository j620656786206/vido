---
name: 'source-command-bmad-bmm-workflows-sprint-planning'
description: 'Generate and manage the sprint status tracking file for Phase 4 implementation, extracting all epics and stories from epic files and tracking their status through the development lifecycle'
---

# source-command-bmad-bmm-workflows-sprint-planning

Use this skill when the user asks to run the migrated source command `bmad-bmm-workflows-sprint-planning`.

## Command Template

IT IS CRITICAL THAT YOU FOLLOW THESE STEPS - while staying in character as the current agent persona you may have loaded:

<steps CRITICAL="TRUE">
1. Always LOAD the FULL @_bmad/core/tasks/workflow.xml
2. READ its entire contents - this is the CORE OS for EXECUTING the specific workflow-config @_bmad/bmm/workflows/4-implementation/sprint-planning/workflow.yaml
3. Pass the yaml path _bmad/bmm/workflows/4-implementation/sprint-planning/workflow.yaml as 'workflow-config' parameter to the workflow.xml instructions
4. Follow workflow.xml instructions EXACTLY as written to process and follow the specific workflow config and its instructions
5. Save outputs after EACH section when generating any documents from templates
</steps>

## MANUAL MIGRATION REQUIRED

Migrated from source command `bmad-bmm-workflows-sprint-planning` into a Codex skill. Invoke it as `$source-command-bmad-bmm-workflows-sprint-planning` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.

Provider automatic file-reference expansion was preserved as text; verify Codex should read those files explicitly.
