---
name: fanloop-maintainer
description: Maintain Fanloop through Define, Build, Certify and Deliver.
---

# Fanloop Maintainer

This directory is one complete Workflow Skill package. The five YAML files beside
this entry define its execution graph; bound Skills and their resources live in
`skills/` beneath this directory.

Use the public `fanloop-workflow` entry and select `fanloop-maintenance`. Initialize
with `fanloop flow init --workflow fanloop-maintainer`, then follow the current
Status and its exact required Skill paths. Do not run this package's steps outside
the Workflow or expose its child Skills globally.
