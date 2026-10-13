# CLI argument source selection (#842)

This bounded design model describes selection of direct payload inputs before
CLI flags, locals and payload assignments are emitted. Transport arguments are
wire occurrences; implicit Basic username/password arguments can name the same
logical payload source. Two explicit operations may have different allocated
locals (`login`, `login2`) but still initialize that same source.

`append` models the current concatenation and fails UniqueDirectSource.
`printed` models deduplication by local spelling and fails on repeated mappings.
`source` keeps the first direct argument for each logical source and passes all
448 states. Both negative configurations exit12 with the named invariant after
136 generated/distinct states; they are not parser failures. All64 subsets of
six bounded candidate operations are explored, including no Basic operations.
An independent body carrier is retained; selection never synthesizes or modifies
argument records. Coverage requires every authored operation's source to have an
input after selection. The model does not claim unique printed names for
*different* logical sources; existing Go name allocation owns that separate duty.

Correspondence: `buildPayloadFieldArgs` retains source names in
AttributeData.Name while allocating VarName for physical operations;
`buildBasicAuthCLIArgs` adds implicit credential operations. Shared CLI
selection `cliPayloadArgs` in `client_cli.go` consumes `cliPayloadInit().ClientArgs` then CLIArgs,
coalescing only records with nonempty FieldName by Name. `makeFlags` and payload
initialization consume the resulting sequence together. HTTP and JSON-RPC share
this owner. No server constructor, wire operation, body decoder or auth scheme
record is removed. Selection must allocate a new slice and leave the input
records intact, including the JSON carrier's pointer identity.

Go correspondence obligations: required/optional/defaulted named credentials,
custom service field names, repeated query/header/cookie operations, path
credentials, retained Basic-only/scalar-body/aggregate-body behavior, and actual
payload plus outbound wire values. The model abstracts away validation, Go type
conversion, defaults, flag-parser absence semantics and JSON body precedence;
compiled runtime tests must cover those. It establishes no new body-vs-direct
input precedence policy. The existing body transformation and subsequent direct
field assignments remain unchanged.

Use Corretto25.0.4.1 with official TLA+tools1.7.4, JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -config source.cfg \
  -metadir /tmp/loom-cli-sources-check CLISources.tla
```

Run append.cfg and printed.cfg separately with fresh state directories. Model
sources and configurations are durable; checker states and raw logs are temporary.
