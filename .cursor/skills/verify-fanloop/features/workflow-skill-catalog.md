# Workflow Skill Catalog

## Sub-features

- Read-only discovery of every legal child Skill, including those not bound to the current Step.
- Absolute package_root, skill_root and SKILL.md paths; stable directory IDs and sorted entries.
- Optional description from YAML frontmatter; absent or unusable metadata produces an empty string without new package restrictions.
- Current Requirement's actual pin selects its offline snapshot or author/live source before caller overrides, with existing failures and no fallback.

## How to get to it (user POV)

After initializing a Requirement, read `fanloop flow skills --help` and run
`fanloop flow skills --root <ABSOLUTE_REQUIREMENT_ROOT>`. Use descriptions and
paths to select further reading when a Skill references another Skill.

## Driving it with fanloop

Use an isolated explicit author source and Requirement. Query from an unrelated cwd and
with `--input '{}'`. Add an unbound Skill with description, a Skill without
frontmatter and one with unusable metadata; all must appear sorted, the latter
two with empty descriptions. Verify a valid YAML string alias resolves its text,
while boolean/map descriptions remain empty. Edit descriptions and call again without rebuilding.
The current Prompt's required list must remain unchanged. Compare State, Output,
Event, Trace and Card hashes around reads; diagnostic cli.jsonl may append.

Reject unknown Request fields, missing Requirement and --dry-run. With a valid actual
pin, set a conflicting or invalid new caller FANLOOP_CONFIG_SOURCE and prove the
catalog still reads the same bound paths and descriptions. Separately break the actual
pin or its selected source; preserve existing error/exit codes and prove no fallback
or business writes. Without an actual pin, invalid-source CLI preflight retains
INTERNAL/exit 5; Runtime bound-load failures may return WORKFLOW_MISMATCH. Use source
copies and separate Roots for mutations.

## Gotchas

Catalog membership does not mean required reading. Use current Status for Step
obligations; read selected catalog paths dynamically. No per-Skill read receipts
or invocation approvals exist. Root SKILL.md is the package entry, not a child
catalog entry. Resources are relative to the chosen Skill's directory.
