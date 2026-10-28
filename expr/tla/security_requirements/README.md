# Per-scheme security requirements

`Requirements.tla` enumerates two schemes, two permission names, every list of
AND branches (including empty/anonymous branches), and every per-scheme grant
set. `checked.cfg` preserves each scheme's required names and passed 4,096
states (8,192 generated) with TLC 1.7.4. `flatten.cfg` deliberately combines
requirements across a conjunction; its initial-state counterexample requires
no grants from scheme `a` and `read` from `b`, but incorrectly requires `read` from `a`.
This compares design candidates; the prior importer explicitly rejected unequal
lists rather than admitting that faulty projection.

Run explicitly from this directory with Java and the installed TLA+ 1.7.4 jar:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config checked.cfg -metadir /tmp/loom-security-requirements Requirements.tla
```

Substitute `flatten.cfg` and a separate temporary `-metadir` for the negative case.
This is an opt-in design tool, never a build, hook, CI or release gate.

`expr.SecurityExpr.SchemeRequirements` owns explicit names by scheme identity.
`RequiredScopes` adds shared declared requirements without crossing scheme
boundaries. `ScopeFor` admits OAuth scopes; `RoleFor` admits non-OAuth roles.
Shared Scope remains a convenience for names required by every declaring scheme;
explicit requirements add to it rather than silently replacing authored intent.
Import uses explicit per-scheme requirements. The application determines grants;
callbacks receive them through the existing RequiredScopes/Validate contract.

Direct DSL/expression tests cover admission, inheritance copies and alternative
identity. The generated scope harness checks exact callback arrays, rejected
roles and grants attached to the wrong scheme. The imported credential fixture
checks wire execution and export. HTTP/gRPC/JSON-RPC projection tests cover shared
service ownership. Wire credential parsing, callbacks, OAuth inventory validity,
list ordering and copying are outside this finite set model and remain Go test
obligations. Keep raw checker output and temporary state out of the repository.
