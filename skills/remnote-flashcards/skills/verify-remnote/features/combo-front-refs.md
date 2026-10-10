# Combo front refs

Claim: each combo card front shows its terms as native refs to the batch's title cards, not literal `[[name]]`.

1. Open `https://www.remnote.com/w/<kb_id>/<combo_rem_id>` using the `knowledgeBaseId` returned by `remnote-cli status`.
2. Wait until no `span.rn-rem-reference` reads `加载中`.
3. In the front line, every planned term is a `span.rem-reference-container[data-rem-reference-id]`, blue, showing the short title. Combine these IDs with the answer IDs and require exact equality with this combo's resolved `preparation_receipt.combo_rem_refs`. No visible `[[` or `]]` text.
4. Hover each ref: the popup has tab `预览` and shows that title card with its back.

Evidence: `<combo_rem_id>-front.png`, `<combo_rem_id>-hover-<ref_id>.png`.
Fail: literal brackets, a ref id not in title IDs (e.g. an alias Rem), missing preview.
