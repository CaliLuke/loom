------------------------- MODULE ResponseAllocation -------------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Mode
VARIABLES semantic, historical, forced, extraBase, authoredOnly, step, names, emitted, definitions
vars == <<semantic, historical, forced, extraBase, authoredOnly, step, names, emitted, definitions>>
Uses == 1..4
Classes == 1..3
OldKeys == 1..2
Groups == {<<s,k>>: s \in Classes, k \in OldKeys}
Group(i) == <<semantic[i], historical[i]>>
Present(g) == \E i \in Uses: Group(i) = g
Base(i) == IF semantic[i] = 3 THEN extraBase ELSE 1
Count(s) == Cardinality({i \in Uses: semantic[i] = s})
OldCount(k) == Cardinality({i \in Uses: historical[i] = k})
Eligible(s) == Count(s) >= 2 \/ \E i \in forced: semantic[i] = s
OldEligible(k) == OldCount(k) >= 2 \/ \E i \in forced: historical[i] = k
Minimum(ns) == CHOOSE n \in ns: \A m \in ns: n <= m
First(g) == Minimum({i \in Uses: Group(i) = g})
NonzeroValues(f) == {f[k]: k \in DOMAIN f} \ {0}

\* Complete parent pass, BEFORE any new semantic eligibility filtering.
RECURSIVE History(_)
History(n) ==
 IF n = 0 THEN [slots |-> [k \in OldKeys |-> 0], owners |-> [k \in OldKeys |-> 0]]
 ELSE LET prev == History(n-1)
          s == semantic[n]
          k == historical[n]
          b == Base(n)
          slot == IF b \notin NonzeroValues(prev.slots) THEN b ELSE b*10+k
      IN IF ~OldEligible(k) \/ prev.slots[k] # 0 THEN prev
         ELSE [slots |-> [prev.slots EXCEPT ![k] = slot],
               owners |-> [prev.owners EXCEPT ![k] = s]]
Old == History(4)
\* Forced usages abstract explicit component-name declarations.
AuthoredReservations == {Base(i): i \in forced} \cup ({authoredOnly} \ {0})
Reserved == NonzeroValues(Old.slots) \cup AuthoredReservations
Used == NonzeroValues(names)
Available(g) == {n \in 1..200: n >= Base(First(g))*10+g[1] /\ n \notin Used
                    /\ (Mode \in {"unreserved", "steal-authored"} \/ n \notin Reserved)
                    /\ (Mode # "steal-authored" \/ n \notin NonzeroValues(Old.slots))}
FirstOld(k) == Minimum({i \in Uses: historical[i] = k})
OwnedKeys(s) == {k \in OldKeys: Old.slots[k] # 0 /\ Old.owners[k] = s}
EarliestOwned(s) ==
 LET ks == OwnedKeys(s) IN CHOOSE k \in ks: \A j \in ks: FirstOld(k) <= FirstOld(j)
UsedBySame(slot,s) == \A h \in Groups: names[h] = slot => h[1] = s
CandidateName(g) ==
 LET slot == Old.slots[g[2]]
     same == {h \in Groups: h[1] = g[1] /\ names[h] # 0}
 IN IF slot = 0 /\ OwnedKeys(g[1]) # {} THEN Old.slots[EarliestOwned(g[1])]
    ELSE IF slot = 0 /\ same # {} THEN names[CHOOSE h \in same: TRUE]
    ELSE IF slot # 0 /\ (Old.owners[g[2]] = g[1] \/ Mode = "steal-owner")
       /\ UsedBySame(slot,g[1]) THEN slot
    ELSE Minimum(Available(g))
CurrentName(g) ==
 LET same == {h \in Groups: h[1] = g[1] /\ names[h] # 0}
     b == Base(First(g))
 IN IF same # {} THEN names[CHOOSE h \in same: TRUE]
    ELSE IF b \notin Used THEN b ELSE b*10+20+g[1]
DroppedReservationName(g) ==
 LET b == Base(First(g)) IN IF b \notin Used THEN b ELSE b*10+g[2]
SelectedName(g) ==
 CASE Mode = "current" -> CurrentName(g)
   [] Mode = "old-dedup" -> Old.slots[g[2]]
   [] Mode = "drop-ineligible" -> DroppedReservationName(g)
   [] OTHER -> CandidateName(g)
Init == /\ semantic \in [Uses -> Classes]
        /\ historical \in [Uses -> OldKeys]
        /\ forced \in SUBSET Uses
        /\ extraBase \in {1, 12}
        /\ authoredOnly \in {0, 13}
        /\ step = 0
        /\ names = [g \in Groups |-> 0]
        /\ emitted = {}
        /\ definitions = [n \in 1..200 |-> 0]
Allocate == /\ step < 4
            /\ LET g == Group(step+1) IN
               names' = IF Eligible(g[1]) /\ names[g] = 0
                        THEN [names EXCEPT ![g] = SelectedName(g)] ELSE names
            /\ step' = step+1
            /\ UNCHANGED <<semantic, historical, forced, extraBase, authoredOnly, emitted, definitions>>
\* Arbitrary map insertion/emission order cannot change preallocated bindings.
Emit(g) == /\ step = 4 /\ names[g] # 0 /\ g \notin emitted
           /\ definitions' = [definitions EXCEPT ![names[g]] = g[1]]
           /\ emitted' = emitted \cup {g}
           /\ UNCHANGED <<semantic, historical, forced, extraBase, authoredOnly, step, names>>
Next == Allocate \/ \E g \in Groups: Emit(g)
Spec == Init /\ [][Next]_vars
TypeOK == /\ step \in 0..4 /\ names \in [Groups -> 0..200]
          /\ emitted \subseteq Groups /\ definitions \in [1..200 -> 0..3]
NonMerge == \A a,b \in Groups: names[a] # 0 /\ names[a] = names[b] => a[1] = b[1]
NoOverwrite == \A g \in emitted: definitions[names[g]] = g[1]
EligibleOnly == \A g \in Groups: names[g] # 0 => Present(g) /\ Eligible(g[1])
Complete == step = 4 => \A g \in Groups: Present(g) /\ Eligible(g[1]) => names[g] # 0
\* Multiple historical slots for one complete contract retain ALL public refs.
OriginalRepresentative == step = 4 => \A k \in OldKeys:
 Old.slots[k] # 0 /\ Eligible(Old.owners[k]) => names[<<Old.owners[k],k>>] = Old.slots[k]
RetiredReservation == \A k \in OldKeys:
 Old.slots[k] # 0 /\ ~Eligible(Old.owners[k]) => Old.slots[k] \notin Used
PublicSlotOwner == \A g \in Groups, k \in OldKeys:
 names[g] # 0 /\ names[g] = Old.slots[k] => g[1] = Old.owners[k] /\ (g[2] = k \/ Old.slots[g[2]] = 0)
UnassignedReuse == step = 4 => \A a,b \in Groups:
 names[a] # 0 /\ names[b] # 0 /\ a[1] = b[1]
 /\ Old.slots[a[2]] = 0 /\ Old.slots[b[2]] = 0 => names[a] = names[b]
AuthoredOnlyReserved ==
 (AuthoredReservations \ NonzeroValues(Old.slots)) \cap Used = {}
=============================================================================
