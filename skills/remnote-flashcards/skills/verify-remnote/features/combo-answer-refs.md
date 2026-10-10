# Combo answer refs

Claim: refs in the combo's STAR answer children (usually the A line, e.g. 点开 X 看…) are native refs to title cards.

1. From the combo page, expand the combo so S/T/A/R children are visible.
2. Wait out `加载中`. Each ref is `span.rem-reference-container[data-rem-reference-id]`. Combine these IDs with the front IDs and require exact equality with this combo's resolved `preparation_receipt.combo_rem_refs`.
3. Hover one ref per answer line: `预览` popup shows the title card back.

Evidence: `<combo_rem_id>-answers.png`, plus one `<combo_rem_id>-answer-hover-<ref_id>.png` per distinct answer ref.
Fail: literal `[[` text, unresolved ref, alias Rem target.
