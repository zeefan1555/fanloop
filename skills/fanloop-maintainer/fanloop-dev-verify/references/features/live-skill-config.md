# Live Skill configuration

## Sub-features

- Workflow-to-Skill group validation.
- Live Skill path resolution on each Status.
- Installed config pointer and explicit `FANLOOP_CONFIG_ROOT` override.

## How to get to it (user POV)

Install Fanloop from a source checkout, then inspect Status `skills[].path`. Skill and exemplar changes become visible from
the configured source without rebuilding the CLI; Workflow semantic changes still require a new release.

## Driving it with fanloop

In an isolated install, record CLI version and first Status Skill path. Make a harmless change only in an isolated copy of a
bound Skill, read Status again, and prove the path/content updates while CLI version and Workflow digest stay unchanged.
Break the isolated config shape and prove Doctor `skill_config` fails closed without fallback.

## Gotchas

Live Skills can affect an in-progress Requirement, while its Workflow digest remains immutable. Every Workflow group must
have a matching Skill directory, Skill IDs are globally unique, and every bound Skill must exist in its own group.
