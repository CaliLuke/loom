# YAML merge ownership investigation

This bounded design model describes libopenapi v0.41.6 model construction,
upstream issue https://github.com/pb33f/libopenapi/issues/657. It establishes only the bounded ownership invariants below, not the dependency's
complete concurrency model.

Two internally scheduled workers can reach one published YAML mapping through
aliases: a components builder and a document lookup. `Begin`/`Expand` abstract
reads by `expandMergeContent`, `mergedNodeContent` and the model builders.
The current `Publish` represents `CheckForMergeNodes` replacing shared
`node.Content` after computing expanded content. TLC must find a write while
the other worker reads. The candidate publication changes only a local pointer
to a merge-expanded shallow view. Its key/value child pointers remain shared
and must stay read-only during construction.

The fix promotes the existing private `utils.resolvedMergeNode` helper to
`ResolveMergeNodes` for use by low-level builders. The expansion algorithm,
local-key precedence, first-merge-source precedence, and alias-cycle detection
remain unchanged. Source nodes retain their authored content. Low-level models
retain effective root views and original key/value nodes, preserving location,
comment and reference metadata. Calls without merges retain node identity.

The model checks absence of overlapping writes, preservation of published
source, effective worker views and eventual completion. It abstracts expansion
as a correct terminating function: Go tests must separately verify local and
sequence precedence, nested aliases, malformed/cyclic input, metadata and
rendering. It does not authorize concurrent application mutation or establish
thread safety of other index/resolver operations.

Run with TLC 1.7.4 (`tla2tools-1.7.4.jar`, SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`)
and Java 25.0.4.1. Use a distinct temporary directory for each invocation to
avoid concurrent standard-module extraction collisions:

```sh
model_tmp=$(mktemp -d)
java -Djava.io.tmpdir="$model_tmp" -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
  tlc2.TLC -workers 1 -metadir "$model_tmp/states" -config Current.cfg MergeOwnership.tla
# Repeat with a fresh directory and Checked.cfg.
```

The current configuration must fail `NoConcurrentWrite`; the checked candidate
must pass all invariants and completion. Preserve summarized counts in this
README after running. Delete temporary model output after review completes.

Current fails `NoConcurrentWrite` after 15 generated / 12 distinct states;
Checked passes 25 generated / 16 distinct states and completion. An initial
malformed Boolean assignment caused a TLC incomplete-state error; that run is
not evidence and was replaced after correcting operator parentheses. Only these summarized results are retained; checker state and raw logs are
not source artifacts.
