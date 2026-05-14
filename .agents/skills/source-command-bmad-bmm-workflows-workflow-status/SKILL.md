---
name: 'source-command-bmad-bmm-workflows-workflow-status'
description: 'Lightweight status checker - answers ""what should I do now?"" for any agent. Reads YAML status file for workflow tracking. Use workflow-init for new projects.'
---

# source-command-bmad-bmm-workflows-workflow-status

Use this skill when the user asks to run the migrated source command `bmad-bmm-workflows-workflow-status`.

## Command Template

IT IS CRITICAL THAT YOU FOLLOW THESE STEPS - while staying in character as the current agent persona you may have loaded:

<steps CRITICAL="TRUE">
1. Always LOAD the FULL @_bmad/core/tasks/workflow.xml
2. READ its entire contents - this is the CORE OS for EXECUTING the specific workflow-config @_bmad/bmm/workflows/workflow-status/workflow.yaml
3. Pass the yaml path _bmad/bmm/workflows/workflow-status/workflow.yaml as 'workflow-config' parameter to the workflow.xml instructions
4. Follow workflow.xml instructions EXACTLY as written to process and follow the specific workflow config and its instructions
5. Save outputs after EACH section when generating any documents from templates
</steps>

## MANUAL MIGRATION REQUIRED

Migrated from source command `bmad-bmm-workflows-workflow-status` into a Codex skill. Invoke it as `$source-command-bmad-bmm-workflows-workflow-status` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.

Provider automatic file-reference expansion was preserved as text; verify Codex should read those files explicitly.
