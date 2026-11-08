----------------------------- MODULE BundleNaming -----------------------------
\* Design-name allocation for bundled components (#957), over the two shared
\* namespaces of the imported design:
\*   T: schema Go type names (assignSchemaNames, internal/openapiimport/names.go)
\*   V: design variables. A schema's variable is always "Imported"+GoName
\*      (render.go:112, render_schema.go:147, plan.go:135) and cannot be
\*      suffixed; security scheme variables are uniqueName("Imported"+
\*      Goify(name)+"Security") (assignSecuritySchemeNames).
\* Keys are lower-cased, as uniqueName compares them; a taken base receives the
\* first free numeric suffix (uniqueName under monotone claims).
EXTENDS Naturals, FiniteSets, Sequences, TLC

CONSTANT Mode
ASSUME Mode \in {"checked", "singlePass", "perNamespace", "syntheticPath", "callerPath", "checkedOrigin"}

\* Schema raw keys in byte order, with lower-cased Goify results (probed):
\*   1 "Foo" -> foo; 2 "FooSecurity" -> foosecurity; 3 "Foo_Security" -> foosecurity
\* Security scheme raw keys in byte order, with their variable bases:
\*   1 "Foo" -> importedfoosecurity; 2 "FooSecurity" -> importedfoosecuritysecurity
SchemaRaw == 1..3
SchemeRaw == 1..2
TBase(n) == IF n = 1 THEN "foo" ELSE "foosecurity"
SBase(m) == IF m = 1 THEN "importedfoosecurity" ELSE "importedfoosecuritysecurity"
Var(g) == "imported" \o g

FirstFree(k, Free(_)) ==
    IF Free(k) THEN k
    ELSE k \o ToString(CHOOSE s \in 2..8 : Free(k \o ToString(s))
                                         /\ \A t \in 2..(s - 1) : ~Free(k \o ToString(t)))

Empty == [T |-> {}, V |-> {}, go |-> [n \in SchemaRaw |-> ""],
          sv |-> [m \in SchemeRaw |-> ""], dup |-> FALSE]

\* One allocation step:
\*   <<"T", n>>  schema type name, checked against T only
\*   <<"J", n>>  external schema type name, chosen so that both the type name and
\*               its fixed variable "Imported"+GoName are free
\*   <<"VS", n>> the schema's fixed variable; claiming a taken one is a duplicate
\*   <<"S", m>>  security scheme variable, checked against V
Step(st, step) ==
    LET kind == step[1] id == step[2] IN
    CASE kind = "T" ->
           LET g == FirstFree(TBase(id), LAMBDA c : c \notin st.T)
           IN [st EXCEPT !.T = @ \cup {g}, !.go[id] = g]
      [] kind = "J" ->
           LET g == FirstFree(TBase(id), LAMBDA c : c \notin st.T /\ Var(c) \notin st.V)
           IN [st EXCEPT !.T = @ \cup {g}, !.V = @ \cup {Var(g)}, !.go[id] = g]
      [] kind = "VS" ->
           [st EXCEPT !.V = @ \cup {Var(st.go[id])}, !.dup = @ \/ Var(st.go[id]) \in st.V]
      [] kind = "S" ->
           LET v == FirstFree(SBase(id), LAMBDA c : c \notin st.V)
           IN [st EXCEPT !.V = @ \cup {v}, !.sv[id] = v]

RECURSIVE Run(_, _)
Run(q, st) == IF q = <<>> THEN st ELSE Run(Tail(q), Step(st, Head(q)))
Asc(S) == [i \in 1..Cardinality(S) |-> CHOOSE n \in S : Cardinality({m \in S : m < n}) = i - 1]
Perms(S) == {f \in [1..Cardinality(S) -> S] : \A i, j \in 1..Cardinality(S) : i # j => f[i] # f[j]}
Steps(kind, seq) == [i \in 1..Len(seq) |-> <<kind, seq[i]>>]

\* Today's entry-only allocation: types by raw name, fixed schema variables,
\* then schemes by raw name.
EntryQueue(es, ek) == Steps("T", Asc(es)) \o Steps("VS", Asc(es)) \o Steps("S", Asc(ek))

\* checked:      complete entry inventory (types, fixed schema variables, scheme
\*               variables), then external schemas by joint choice, then external
\*               schemes; external orders are arbitrary permutations.
\* singlePass:   today's code over the union.
\* perNamespace: the rejected revision 4 rule: two passes per namespace, but all
\*               fixed schema variables are reserved before any scheme variable.
Queues(es, xs, ek, xk) ==
    CASE Mode \in {"checked", "syntheticPath", "callerPath", "checkedOrigin"} ->
           {EntryQueue(es, ek) \o Steps("J", p) \o Steps("S", q) : p \in Perms(xs), q \in Perms(xk)}
      [] Mode = "singlePass" ->
           {Steps("T", Asc(es \cup xs)) \o Steps("VS", Asc(es \cup xs)) \o Steps("S", Asc(ek \cup xk))}
      [] Mode = "perNamespace" ->
           {Steps("T", Asc(es)) \o Steps("T", p) \o Steps("VS", Asc(es \cup xs))
              \o Steps("S", Asc(ek)) \o Steps("S", q) : p \in Perms(xs), q \in Perms(xk)}

\* The allocator previously assumed declaration ownership was already correct.
\* Schema promotion must retain authored paths, resetting at a reference target.
OriginCases == {"entryInline", "externalInline", "externalNested",
                "entryRefExternal", "externalRefEntry"}
SourcePath(c) == IF c \in {"externalInline", "externalNested", "externalRefEntry"}
                 THEN <<"External">> ELSE <<"Entry">>
TargetPath(c) == CASE c = "entryRefExternal" -> <<"External">>
                  [] c = "externalRefEntry" -> <<"Entry">>
                  [] OTHER -> SourcePath(c)
AuthoredPath(c) == TargetPath(c) \o
                   (IF c = "externalNested" THEN <<"oneOf", "property", "oneOf", "node">>
                    ELSE <<"oneOf", "node">>)
ProjectedPath(c) ==
    CASE Mode = "syntheticPath" /\ c \in {"entryInline", "externalInline", "externalNested"}
         -> <<"GeneratedVariantNode">>
      [] Mode = "callerPath" -> SourcePath(c) \o <<"node">>
      [] OTHER -> AuthoredPath(c)
\* Unrecognized synthetic paths fall back to entry ownership, as
\* bundledDescription.declarationOrigin does after its ancestor lookup.
ExternalOwner(path) == path[1] = "External"

VARIABLES es, xs, ek, xk, queue, st, originCase
vars == <<es, xs, ek, xk, queue, st, originCase>>

Init ==
    /\ originCase \in IF Mode \in {"syntheticPath", "callerPath", "checkedOrigin"}
                       THEN OriginCases ELSE {"entryInline"}
    /\ es \in SUBSET SchemaRaw
    /\ xs \in SUBSET (SchemaRaw \ es)
    /\ ek \in SUBSET SchemeRaw
    /\ xk \in SUBSET (SchemeRaw \ ek)
    /\ queue \in Queues(es, xs, ek, xk)
    /\ st = Empty

Allocate == queue # <<>> /\ st' = Step(st, Head(queue)) /\ queue' = Tail(queue)
            /\ UNCHANGED <<es, xs, ek, xk, originCase>>
Done == queue = <<>> /\ UNCHANGED vars
Next == Allocate \/ Done
Spec == Init /\ [][Next]_vars

\* Oracle: the names an entry-only import allocates today.
EntryAlone == Run(EntryQueue(es, ek), Empty)
EntryStable ==
    queue = <<>> => /\ \A n \in es : st.go[n] = EntryAlone.go[n]
                    /\ \A m \in ek : st.sv[m] = EntryAlone.sv[m]
\* Schema type names alone (the earlier schema-only property).
EntryTypesStable == queue = <<>> => \A n \in es : st.go[n] = EntryAlone.go[n]
\* Witness (expected "violated"): the final names for the reviewed pair, entry
\* security scheme "Foo" with external schema "FooSecurity".
PairWitness == ~(queue = <<>> /\ es = {} /\ xs = {2} /\ ek = {1} /\ xk = {})
\* No two declarations share a type name or a design variable.
NoDuplicate ==
    queue = <<>> => /\ ~st.dup
                    /\ \A n1, n2 \in es \cup xs : n1 # n2 => st.go[n1] # st.go[n2]
                    /\ \A n \in es \cup xs, m \in ek \cup xk : Var(st.go[n]) # st.sv[m]
                    /\ \A m1, m2 \in ek \cup xk : m1 # m2 => st.sv[m1] # st.sv[m2]
OriginPreserved == ExternalOwner(ProjectedPath(originCase)) = ExternalOwner(AuthoredPath(originCase))
=============================================================================
