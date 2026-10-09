# FlashcardsPlan

`remnote-cli flashcards apply` takes a plan file. The live field list is the installed CLI: `remnote-cli flashcards apply --help`, and the RemNote Direct source `packages/cli/src/schemas/task-schemas.ts`. This note records the shape the plugin is organized around. It is not a sample of anyone's knowledge base. Do not fill it with invented Rem IDs, titles, or card text.

```text
FlashcardsPlan = {
  schemaVersion: 1,
  dailyRemId,
  homeRootId,
  tagRemId,
  title?,
  cardCluster?,
  newCards?[<=7] {
    key,
    question,
    answer: Segment[][]
  },
  answerPatches?,
  reusedCardRemIds?
}

Segment = string | { remId }
```

Rules that sit on this shape:

- `schemaVersion` is `1`.
- `newCards` has at most 7 entries. A plan is also at most 100 KiB. More independent memory goals are another plan, not a larger cap and not a truncated answer.
- `question` is a plain string. It does not parse brackets.
- An answer `Segment` is either a string or `{ "remId": "<title card Rem ID>" }`. That object is how a combo card points at a title card inside `flashcards apply`. Do not put `pageKey` in a flashcards plan.
- `cardCluster` is omitted or `false` unless the user explicitly asked for a cluster and the learning-card contract allows `true`.
- `answerPatches` and `reusedCardRemIds` are optional. Use a patch only with the exact card or answer ID and the `expectedTextHash` just read. The precise patch object is whatever `--help` shows for this install. Neither field moves a card under a different parent.
- `dailyRemId`, `homeRootId`, and `tagRemId` come from this connection's readback, not from this file. Look up today's daily note with a read-only command first. `sdk date get-todays-doc` is a write: it creates the note when it is missing. A Chinese `search` that returns nothing is not proof the note or tag is absent.
- There is no field that selects an existing theme. `title`, when present, creates one new theme Rem under `dailyRemId` and parents this plan's `newCards` under that new theme. A later plan with `title` creates another theme. The combo plan omits `title`. After that apply the combo's parent is the daily note. Move it to the last child of the theme, then remove its own tag. Do not leave the combo on the daily note.

```text
remnote-cli move-note <COMBO_ID> --new-parent-rem-id <THEME_ID> --position last --expected-old-parent-rem-id <DAILY_ID>
remnote-cli move-note <COMBO_ID> --new-parent-rem-id <THEME_ID> --position last --expected-old-parent-rem-id <DAILY_ID> --apply
remnote-cli sdk rem object-remove-tag --target-id <COMBO_ID> --args-json '["<TAG_ID>", false]' --allow-destructive
```

`update-tags --remove-tag-ids` fails because it requires allowDestructive. One command per shell. Dry-run `move-note` first, then the same command with `--apply`. Read back theme child order (combo last), empty combo tags, `review-stats --root-id <THEME_ID>` listing every card, and `search-by-tag` returning only the theme. `newCards[].key` is not a Segment, so `{ "remId" }` may name only a Rem that already exists.

Question and answer lines, not a markdown import sketch. Placeholders only. The titled plan is the term cards. The combo plan omits `title` and may cite a title Rem ID that already exists:

```text
{
  schemaVersion: 1,
  dailyRemId: "<today's daily note Rem ID>",
  homeRootId: "<HOME Rem ID>",
  tagRemId: "<Concept Rem ID already read back>",
  newCards: [
    {
      key: "pe-gift",
      question: "<combo question>",
      answer: [
        ["S — Situation：背景"],
        ["..."],
        [{ "remId": "<title Rem ID that already exists>" }]
      ]
    }
  ]
}
```

`newCards[].key` is not a Rem ID. Aliases are not a plan field. After the title card exists, one call per card: `remnote-cli sdk rem object-get-or-create-alias-with-text --target-id <ID> --args-json '[["<alias>"]]'`. `update --add-aliases` depends on the SDK version. On SDK 0.0.46 it fails with `Native transactions are unavailable: SDK 0.0.46 cannot acknowledge abort`.

The idempotency key is a flag, not a plan field: `flashcards apply --plan <FILE|-> --idempotency-key <KEY>`. Preview is the default. A preview without `--apply` reports `validated` and `planHash` only, not the cards. Read the plan file and compare it to the reviewed draft before a separate `--apply` invocation.
