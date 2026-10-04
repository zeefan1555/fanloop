---
status: accepted
date: 2026-10-04
amends: ADR-0040, ADR-0050, ADR-0124-load-local-workflow-skill-packages
supersedes_in_part: ADR-0110, ADR-0118
---

# Discover the Complete Workflow Skill Catalog

`fanloop flow skills` is a read-only method for an existing Requirement. The empty
Request resolves the Requirement's Workflow ID through the same current local
source loading as Status. Its response includes Workflow reference, absolute
package/Skill roots and every direct child Skill's directory ID, optional
frontmatter description (empty when absent or unusable), and absolute SKILL.md
path. Entries sort by ID. A legal Skill need not acquire metadata to remain legal.

The catalog is navigation, independent of current Step bindings. Agents use the
entry Skill, task and Skill references to choose additional reading. No reading
receipt, execution enforcement, task scheduler, shared Skill provider or remote
configuration mechanism is introduced. Status, State/Event/Output, YAML schema,
source failure behavior and fallback prohibition are unchanged. Read calls may
write existing best-effort diagnostic transcripts but no business facts.

The public method and DTOs live in idl/flow.thrift and use the existing generated
Service/validator/CommandSpec chain. Leaf Help supplies the empty Request and
navigation contract. The command generator derives inventory from Thrift rather
than duplicating a fixed public-command count.

Delete the entire pstack-native package, its scenario, active inventory and
dedicated bootstrap tests. Retain historical ADRs; this decision supersedes
ADR-0118's production choice and only pstack-native inventory claims in ADR-0110
and ADR-0124. The retained packages are fanloop-maintainer, remnote-flashcards and
remnote-direct-maintainer. No replacement pstack Workflow is approved here.

Approval: the user approved adding the catalog and deleting pstack-native, then
approved the concrete flow.thrift proposal SHA256
19410a92672cc1b035df957b32beeae3d48eee98b15b3f7e71fc97f315732655 on 2026-10-04.
The proposal only adds FlowSkillsRequest, SkillCatalogEntry, FlowSkillsResponse
and FlowService.Skills. Delivery stops at one PR; merge and actual installation
remain outside the authorized scope.
