# Live Workflow configuration

## Sub-features

- Five-file Workflow bundle and bound Skill validation from one live root.
- Live Prompt and Skill path resolution on every CLI invocation.
- Stable installed tree at `current/workflows` with no Release fallback.

## How to get to it (user POV)

Install Fanloop, then inspect `current/workflows` and Status `skills[].path`. Valid Workflow YAML and Skill edits become
visible on the next command without rebuilding or reinstalling the CLI.

## Driving it with fanloop

In an isolated install, run `flow init` and `flow status`, edit the live `prompt.yaml` and a bound `SKILL.md`, then run
Status again and prove the new Prompt plus the same-root absolute Skill path are returned. Break the live YAML or Skill
shape and prove `flow status` plus `verify doctor` fail closed without fallback.

## Gotchas

Live edits affect an in-progress Requirement on its next invocation. Every Workflow directory must contain exactly the
five YAML files plus `skills/`; symlinked groups/files, empty Skills, missing bindings, and incompatible current State fail.
