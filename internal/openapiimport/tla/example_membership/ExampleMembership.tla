---- MODULE ExampleMembership ----
\* Example membership phase and authored ownership. Components are analyzed in
\* declaration order; promoted constructors join the final component set after
\* analysis. Each component may carry local assertions, reference another
\* component, and carry one example. Normalization may flatten the component
\* into a nullable wrapper (wrap); the example is authored either on the inner
\* non-null branch or on the wrapper occurrence (author). Mode selects when
\* membership is decided, which registry resolves references, and which domain
\* judges a copied example.
EXTENDS FiniteSets, Naturals, Sequences

CONSTANTS Comp, Values, Null, NoEx, None, Missing, Mode

ASSUME Mode \in {"legacy", "readtime", "stale", "reffirst", "container", "final"}

Targets == Comp \cup {None, Missing}

VARIABLES local, target, ex, wrap, author, promoted, order, step, retained, done

vars == <<local, target, ex, wrap, author, promoted, order, step, retained, done>>

N == Cardinality(Comp)

\* Declared components precede promoted constructors, as in analyzer.document.
Orders(P) ==
    {f \in [1..N -> Comp] :
        /\ \A i, j \in 1..N : i # j => f[i] # f[j]
        /\ \A i, j \in 1..N : f[i] \in P /\ f[j] \notin P => j < i}

\* Final normalized semantics of the inner branch: the conjunction of local
\* assertions and the reference target. A missing target or a cycle at the same
\* value position is never membership. registry resolves references.
RECURSIVE Conj(_, _, _, _)
Conj(c, v, seen, registry) ==
    /\ c \notin seen
    /\ v \in local[c]
    /\ \/ target[c] = None
       \/ /\ target[c] \in registry
          /\ Conj(target[c], v, seen \cup {c}, registry)

\* The schema that renders the example: the wrapper admits null when wrap holds.
Rendering(c, v, registry) == IF v = Null THEN wrap[c] ELSE Conj(c, v, {}, registry)

\* The domain the example was authored against.
Authored(c, v, registry) ==
    IF author[c] = "inner" THEN v # Null /\ Conj(c, v, {}, registry) ELSE Rendering(c, v, registry)

Member(c, v) == Authored(c, v, Comp) /\ Rendering(c, v, Comp)

\* Pre-fix matcher: the reference returns first and local assertions are
\* ignored; an unregistered or missing target is accepted.
RECURSIVE Legacy(_, _, _, _)
Legacy(c, v, seen, known) ==
    IF c \in seen THEN FALSE
    ELSE IF target[c] = None THEN v \in local[c]
    ELSE IF target[c] \in known THEN Legacy(target[c], v, seen \cup {c}, known)
    ELSE TRUE

\* Pre-fix read time judged the authored occurrence before any copy.
LegacyAuthored(c, v, known) ==
    IF v = Null THEN author[c] = "outer" /\ wrap[c] ELSE Legacy(c, v, {}, known)

\* a.schemas while analyzing order[step]: declared components seen so far.
Known == {order[i] : i \in 1..step} \ promoted

Init ==
    /\ local \in [Comp -> SUBSET Values]
    /\ target \in [Comp -> Targets]
    /\ ex \in [Comp -> Values \cup {Null, NoEx}]
    /\ wrap \in [Comp -> BOOLEAN]
    /\ author \in [Comp -> {"inner", "outer"}]
    /\ promoted \in SUBSET Comp
    /\ order \in Orders(promoted)
    /\ step = 1
    /\ retained = {}
    /\ done = FALSE

\* Read phase. legacy and readtime decide membership now against the partial
\* registry; every other mode only admits and defers.
Read ==
    /\ step <= N
    /\ LET c == order[step] IN
       retained' =
         IF ex[c] = NoEx THEN retained
         ELSE CASE Mode = "legacy" -> IF LegacyAuthored(c, ex[c], Known) THEN retained \cup {c} ELSE retained
              [] Mode = "readtime" -> IF Authored(c, ex[c], Known) THEN retained \cup {c} ELSE retained
              [] OTHER -> retained \cup {c}
    /\ step' = step + 1
    /\ UNCHANGED <<local, target, ex, wrap, author, promoted, order, done>>

\* Membership phase over the complete graph. stale resolves through the
\* analysis registry, which never contains promoted constructors. reffirst has
\* the final timing and authored binding but the pre-fix reference-first
\* matcher. container judges a copied example only by the schema that now
\* carries it, keeping its path but not its authored domain.
Validate ==
    /\ step > N
    /\ ~done
    /\ retained' =
         CASE Mode = "final" -> {c \in retained : Member(c, ex[c])}
         [] Mode = "stale" -> {c \in retained : Authored(c, ex[c], Comp \ promoted) /\ Rendering(c, ex[c], Comp \ promoted)}
         [] Mode = "reffirst" -> {c \in retained : LegacyAuthored(c, ex[c], Comp)}
         [] Mode = "container" -> {c \in retained : Rendering(c, ex[c], Comp)}
         [] OTHER -> retained
    /\ done' = TRUE
    /\ UNCHANGED <<local, target, ex, wrap, author, promoted, order, step>>

\* The analysis has finished; the final document no longer changes.
Finished == done /\ UNCHANGED vars

Next == Read \/ Validate \/ Finished

Spec == Init /\ [][Next]_vars

\* Every retained example belongs to its authored owner and its rendering schema.
Sound == done => \A c \in retained : Member(c, ex[c])

\* Every authored member example is retained.
Complete == done => \A c \in Comp : ex[c] # NoEx /\ Member(c, ex[c]) => c \in retained
====
