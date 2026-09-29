--------------------- MODULE RepresentationEquivalence ---------------------
EXTENDS Naturals, FiniteSets, Sequences

CONSTANTS NodeCount, SlotCount, Mode
VARIABLES labels, edges, classes, round
vars == <<labels, edges, classes, round>>
Nodes == 1..NodeCount
Slots == 1..SlotCount
Labels == {0, 1}

\* Labels abstract exact non-reference schema data, including annotations and
\* external reference text. Slots abstract ordered local-reference paths.
\* Zero denotes absence of a local reference at that slot.
Skeleton(n) == <<labels[n], [s \in Slots |-> edges[n][s] = 0]>>
Partition(keys) ==
    [n \in Nodes |->
        CHOOSE r \in Nodes :
            /\ keys[r] = keys[n]
            /\ \A m \in Nodes : keys[m] = keys[n] => r <= m]
Refined ==
    Partition([n \in Nodes |->
        <<classes[n], labels[n],
          [s \in Slots |->
            IF edges[n][s] = 0 THEN 0 ELSE classes[edges[n][s]]]>>
    ])
ClassCount(partition) == Cardinality({partition[n] : n \in Nodes})
Stable == ClassCount(Refined) = ClassCount(classes)

Init ==
    /\ labels \in [Nodes -> Labels]
    /\ edges \in [Nodes -> [Slots -> (Nodes \cup {0})]]
    /\ classes = Partition([n \in Nodes |-> Skeleton(n)])
    /\ round = 0

RefineStep ==
    /\ ~Stable
    /\ round < NodeCount
    /\ classes' = Refined
    /\ round' = round + 1
    /\ UNCHANGED <<labels, edges>>

Spec == Init /\ [][RefineStep]_vars

\* Independent finite observation uses no partition or traversal-cycle markers.
\* NodeCount + 1 levels distinguish this bounded deterministic graph domain.
RECURSIVE Observe(_, _)
Observe(n, depth) ==
    IF n = 0 THEN <<"absent">>
    ELSE IF depth = 0 THEN <<"opaque">>
    ELSE <<"node", labels[n], [s \in Slots |-> Observe(edges[n][s], depth - 1)]>>
Equivalent(a, b) == Observe(a, NodeCount + 1) = Observe(b, NodeCount + 1)

Position(value, sequence) ==
    CHOOSE i \in 1..Len(sequence) : sequence[i] = value
Members(sequence) == {sequence[i] : i \in 1..Len(sequence)}

\* Legacy recursion uses original component identity and active DFS depth.
\* Hashing is abstracted by exact nested tuples; hash collisions are excluded.
RECURSIVE Legacy(_, _)
Legacy(n, active) ==
    IF n = 0 THEN <<"absent">>
    ELSE IF n \in Members(active) THEN <<"recursive", Position(n, active) - 1>>
    ELSE <<"node", labels[n],
           [s \in Slots |-> Legacy(edges[n][s], Append(active, n))]>>

\* Candidate recursion uses only stabilized quotient classes. Representatives
\* are conveniences and never appear in the returned fingerprint.
RECURSIVE Quotient(_, _)
Quotient(n, active) ==
    IF n = 0 THEN <<"absent">>
    ELSE LET representative == classes[n]
         IN IF representative \in Members(active)
            THEN <<"recursive", Position(representative, active) - 1>>
            ELSE <<"node", labels[representative],
                   [s \in Slots |->
                     Quotient(edges[representative][s],
                              Append(active, representative))]>>
Fingerprint(n) ==
    IF Mode = "legacy" THEN Legacy(n, <<>>)
    ELSE IF Mode = "overmerge" THEN <<labels[n]>>
    ELSE Quotient(n, <<>>)

TypeOK ==
    /\ labels \in [Nodes -> Labels]
    /\ edges \in [Nodes -> [Slots -> (Nodes \cup {0})]]
    /\ classes \in [Nodes -> Nodes]
    /\ round \in 0..NodeCount
RefinementOnlySplits ==
    \A a, b \in Nodes : Refined[a] = Refined[b] => classes[a] = classes[b]
ConvergesWithinBound == round = NodeCount => Stable
PartitionMatchesObservation ==
    Stable => \A a, b \in Nodes : (classes[a] = classes[b]) = Equivalent(a, b)
EquivalentContractsShare ==
    Stable => \A a, b \in Nodes :
        Equivalent(a, b) => Fingerprint(a) = Fingerprint(b)
DifferentContractsStaySeparate ==
    Stable => \A a, b \in Nodes :
        Fingerprint(a) = Fingerprint(b) => Equivalent(a, b)

\* Compatibility requires retaining the existing fingerprint serialization.
\* This checks its abstract structure; exact JSON/hash bytes remain a Go test.
RECURSIVE Acyclic(_, _)
Acyclic(n, active) ==
    IF n = 0 THEN TRUE
    ELSE IF n \in active THEN FALSE
    ELSE \A s \in Slots : Acyclic(edges[n][s], active \cup {n})
AcyclicFingerprintsPreserved ==
    Stable => \A n \in Nodes :
        Acyclic(n, {}) => Quotient(n, <<>>) = Legacy(n, <<>>)
=============================================================================
