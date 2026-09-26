# Published Workflow configuration

## Sub-features

- Fornax Online/prod release selection, exact-version metadata and complete Workflow package download.
- Safe archive paths/types/size, same-root bound Skills and complete package resources.
- Explicit update-only full-tree replacement; remote preparation does not hold the configuration lock, and final activation is atomic under a short exclusive lock.

## How to get to it (user POV)

Configure fornax.json with workspace and selected Workflow id/key/skill_id. Publish a complete Skill package in Fornax, then run `fanloop update`. All other commands and local installation stay offline.

## Driving it with fanloop

Use the isolated installed candidate. With explicit authorized Fornax credentials, run `fanloop update`, then flow init/status and compare every
file against the reviewed fixed publication manifest. Separately use a deterministic external CLI fixture to exercise a
newer draft, published update, removed resource, malformed archive, missing release and identity mismatch. Prove failures
leave State/Event and live bytes unchanged. Disable Fornax and prove flow/status/doctor/dashboard and local reinstall still work from the retained tree. Pause the remote helper and prove ordinary shared reads remain available, then run concurrent activation reads and prove each reads one complete tree.
Use scripts/validate-configuration.py for the fixed production Route Matrix; it does not execute business Skill actions.

## Gotchas

Key-only Fornax lookup can return an unreleased commit. Always query the released version explicitly. Each Workflow is one Fornax Skill with its entire YAML/Skill/resource tree.
FANLOOP_CONFIG_SOURCE is only an explicit candidate/verification source; never the global live root or a failure fallback.
