# Workflow products

## Sub-features

- `technical-solution-design` document production and human gates.
- `material-flashcards` recall-goal planning, preview approval and Decks persistence.
- `fanloop-maintainer` implementation, review, Agent acceptance, human acceptance, PR merge and local activation.

## How to get to it (user POV)

Select the matching scenario through `fanloop-workflow`, initialize the returned Workflow ID and follow each Status Prompt
and Skill path. The product is the observable artifacts and governed transitions, not only terminal completion.

## Driving it with fanloop

Use a fresh Root per Workflow. Verify the first meaningful user artifact, at least one forward transition, the relevant Card
panorama and the nearest governance boundary. Stop at Human Steps unless a new, explicit human decision is part of the
current test. Compare created files and Status Outputs with the Workflow's public descriptions.

## Gotchas

Workflow Steps, Conditions and Routes evolve independently. Read the bound five-file bundle and current Status rather than
copying historical counts. Workflow-specific behavior must remain in YAML and Skills, not in generic verify Runtime code.
