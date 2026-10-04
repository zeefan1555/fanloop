---
status: accepted
date: 2026-10-04
amends: ADR-0109, ADR-0123, ADR-0125
---

# Main Agent Decides Within the Authorized Scope

The main Agent defaults to deciding requirements, concrete contracts (including
Thrift IDL), candidate acceptance and integration within the user's authorized
goal, scope, side effects and stopping point. It reviews the concrete material
first and records an explicit decision. An execution child must request and wait
for that decision; permission to implement is not a substitute for contract review.

Ask the user only for a specifically retained human gate, a decision outside the
authorized scope or information only a human can provide. A user's chosen delivery
endpoint and external-write boundaries remain binding. Current Workflow
conditions still require real evidence and cannot be edited to fabricate progress.

Root AGENTS.md is the operative governance source. Thrift remains the contract
truth; before/after snippets, field IDs, optionality, annotations, errors, consumer
impact, alternatives and an explicit main Agent review are required before edits.
Generation, independent Code Review/public CLI blackbox, CI and exact-candidate
checks are preserved. Historical human-review records remain historical facts;
this changes the current default decision authority, not those records.

The user explicitly requested this AGENTS.md change and said “你全程都可以决定，
我同意了” on 2026-10-04 (host message
01a1034a-6ebf-7d12-bf49-607f28d6514e). The main Agent approved this Requirement's
scope extension and exact IDL; autonomous decisions do not enlarge the existing
stop-at-PR boundary.
