# OAuth flow inventories

`Inventories.tla` checks two flows and two scope names with shared defaults,
flow-local inventories, and an explicit-block flag. `FlowScopes` in
`expr/security_scopes.go` implements the selected inventory. A supplied empty
block overrides defaults; a missing block inherits them. `ScopeNames` computes
availability for requirement admission, never a replacement flow inventory.

Run explicitly with a locally installed TLA+ 1.7.4 jar:

```sh
java -cp "$TLA_JAR" tlc2.TLC -config checked.cfg Inventories.tla
java -cp "$TLA_JAR" tlc2.TLC -config flatten.cfg Inventories.tla
```

The checked model explores 256 distinct states (512 generated) and preserves
Isolation. The flattening candidate fails initially: an empty inherited flow
receives a name declared only by another flow. These are opt-in design checks.
No proof tools are required by ordinary builds, hooks or CI.

The model abstracts descriptions, URLs and authentication policy. Expression
and DSL tests cover inheritance, explicit empty inventories and deep copies;
import round trips and rendered OpenAPI tests cover exact names/descriptions;
the generated security fixture checks callback inventories, required grants
and denial. Authentication, issuer validation and token grants remain owned
by the application. The finite model does not prove the Go implementation.
