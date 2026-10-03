---
name: fanloop-maintainer
description: Maintain Fanloop code and mixed implementation through Define, Build, Certify and Deliver.
---

# Fanloop Maintainer

This directory is one complete Workflow Skill package. The five YAML files beside
this entry define its execution graph; bound Skills and their resources live in
`skills/` beneath this directory.

For pure Workflow configuration or Skill text, read the repository's
`.agents/skills/edit-fanloop-skills/SKILL.md` and deliver directly without initializing
this Workflow. Business Workflow validation and approval requirements still apply.

For code and mixed implementation, use the public `fanloop-workflow` entry and select `fanloop-maintenance`. Initialize
with `fanloop flow init --workflow fanloop-maintainer`, then follow the current
Status and its exact required Skill paths. Do not run this package's steps outside
the Workflow or expose its child Skills globally.

Code and mixed implementation retain full certification. Each Step of this Workflow
still requires its real Panorama; pure configuration maintenance has no such Steps.
