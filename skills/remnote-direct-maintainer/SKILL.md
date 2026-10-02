---
name: remnote-direct-maintainer
description: Maintain RemNote Direct through impact-based checks, exact-candidate acceptance, Git delivery and local adoption.
---

# RemNote Direct Maintenance

This is a complete repository-local Workflow package, not a build or release framework.
Use it for an explicitly authorized change to the independent CLI/plugin repository;
ordinary card creation uses `remnote-flashcards`.

Resolve the Requirement root and run `fanloop flow status --root <ROOT>` first.
Only after NOT_INITIALIZED, initialize with
`fanloop flow init --root <ROOT> --workflow remnote-direct-maintainer --title <TITLE>`.
The explicit workflow ID is this package's entry; it does not need a new alias in
an installed public Skill's static scenario menu. Do not rebuild Fanloop to load it.

Follow current Status, its exact Skill paths, and configured routes. Each step
displays its own Panorama once. Repository rules supply the source path, commands,
Git destination, supported installation method and authorized test environment.
