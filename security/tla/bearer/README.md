# Bearer credential boundary (#607)

Credential constraints belong to the extracted value. Authorization mappings for
JWT/OAuth use bearer framing; custom mappings and API keys preserve raw values.
This follows the documented distinction between HTTP bearer and API-key-style
mappings. Permissive post-construction prefix guessing is removed, including
acceptance of raw/other-scheme Authorization and stripping custom/API-key values.
See accepted/rejected examples in docs/dsl-reference.md.

The legacy control validates the framed value and violates RoundTrip for
`Bearer valid`. Checked extraction followed by validation passes Contract and
RoundTrip (16 distinct states). This model abstracts strings to four examples
and constraints to one accepted value. It does not prove the parser, Unicode,
Go pointer representation, multiple headers, authentication, or transport code.

Run with TLA+ tools v1.7.4 (TLC 2.19 rev `5a47802`), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Validated with Corretto 25.0.4.1. From this directory:

```sh
for cfg in Legacy Checked; do
  java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
    -metadir "/tmp/loom-bearer-$cfg" -config "$cfg.cfg" Bearer.tla
done
```

Legacy must fail RoundTrip (exit 12); Checked must pass (exit 0). Parser errors
are not negative-control evidence. Delete checker outputs after review.

Implementation and tests:

- `internal/securitygen.IsBearer` classifies the actual wire binding, matching
  the credential attribute rather than unrelated schemes on the same method.
- `security.DecodeBearer` parses only bearer framing. HTTP/gRPC decode their
  string local before validation and preserve optionality and named types when
  constructing the service payload. Client encoders always add bearer framing.
- `TestBearerCredentials` compiles generated HTTP and gRPC transports for 16
  combinations of JWT/OAuth, plain/named strings, required/optional presence and
  Authorization/custom mappings. It exercises valid/invalid/missing values,
  protocol errors, casing, opaque API keys and HTTP dispatch rejection. It also
  reproduces and protects the optional named-metadata length-validation panic.
- `TestDecodeBearer` and `TestBearerMapping` pin parsing and classification.
  Existing named credential and Basic-auth tests cover unchanged auth callbacks.

Consumer inspection found the checked-in quality fixture uses BearerTransport;
the sibling loom-mcp design/test references do not establish a reason to accept
unframed Authorization. No consumer-specific exceptions are added.
