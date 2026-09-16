# Multi-surface journeys

## Sub-features

- Candidate install through Requirement initialization and first transition.
- Flow facts reflected in Trace, Card and Doctor.
- Candidate/config drift, recovery and evidence cleanup.

## How to get to it (user POV)

Start from a source candidate, install it in isolation, choose a Workflow, drive the Requirement, inspect its projections,
capture it with `fanloop verify snapshot --root ROOT`, then remove only run-owned state with
`fanloop verify cleanup --run-id RUN_ID`. This is the broad regression path after individual Feature checks.

## Driving it with fanloop

Record source HEAD and global current, install the candidate, run version/Doctor, initialize a fresh Root, render Card,
submit one dry-run and real transition, render Trace, rerun Doctor and capture all artifacts. Change only an isolated live
Skill copy and prove Status refreshes it without Workflow drift. Clean the session and verify evidence and global current.

## Gotchas

One green surface cannot stand in for the journey: tests do not prove installation, Card does not prove delivery, and final
Status does not prove the submitted action. Keep every candidate, Root, config source and evidence bundle identity aligned.
