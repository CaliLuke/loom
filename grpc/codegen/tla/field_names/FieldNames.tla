----------------------------- MODULE FieldNames -----------------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Mode
Slots == {"reset", "label", "getlabel", "getpick", "branch", "oneof", "mapbranch", "reflect"}
Count == Cardinality(Slots)
Empty == <<0, "empty", 0>>
Name(prefix, stem, suffix) == <<prefix, stem, suffix>>
Getter(n) == Name(n[1] + 1, n[2], n[3])
Suffix(n, i) == Name(n[1], n[2], n[3] + i)
Reserved == {Name(0, "reset", 0)}
ActualReserved == Reserved \cup {Name(0, "reflect", 0)}
NestedTypes == {Name(0, "mapentry", 0)}
Branches == {"branch", "mapbranch"}
Base(s, version) == CASE s = "reset" -> Name(0, "reset", 0)
                     [] s = "label" -> Name(0, "label", 0)
                     [] s = "getlabel" -> Name(1, "label", 0)
                     [] s = "getpick" -> Name(1, "pick", 0)
                     [] s = "branch" -> Name(0, "code", 0)
                     [] s = "oneof" -> Name(0, "pick", version)
                     [] s = "reflect" -> Name(0, "reflect", version)
                     [] OTHER -> Name(0, "mapentry", 0)
Orders == {o \in [1..Count -> Slots]:
             {o[i]: i \in 1..Count} = Slots /\
             \E i \in 1..(Count-1): o[i] = "branch" /\ o[i+1] = "oneof"}
Available(n, used, getter) == n \notin used /\ (~getter \/ Getter(n) \notin used)
Unique(n, used, getter) ==
  LET i == CHOOSE k \in 0..Count:
                Available(Suffix(n, k), used, getter) /\
                \A j \in 0..(k-1): ~Available(Suffix(n, j), used, getter)
  IN Suffix(n, i)
VARIABLES order, index, version, used, pb, loom, wrappers, pbWrappers
vars == <<order, index, version, used, pb, loom, wrappers, pbWrappers>>
Init == /\ order \in Orders
        /\ index = 1 /\ version = 0 /\ used = Reserved
        /\ pb = [s \in Slots |-> Empty] /\ loom = pb
        /\ wrappers = pb /\ pbWrappers = pb
Step ==
  /\ index <= Count
  /\ LET s == order[index]
         raw == Base(s, version)
         getter == s # "oneof"
         name == Unique(raw, used, getter)
         wrapper == IF s \in Branches THEN Unique(name, NestedTypes, FALSE) ELSE Empty
     IN /\ pb' = [pb EXCEPT ![s] = name]
        /\ loom' = [loom EXCEPT ![s] = IF Mode = "legacy" THEN raw ELSE name]
        /\ pbWrappers' = [pbWrappers EXCEPT ![s] = wrapper]
        /\ wrappers' = [wrappers EXCEPT ![s] =
             IF s \notin Branches THEN Empty
             ELSE IF Mode = "checked" THEN wrapper
             ELSE IF Mode = "legacy" THEN raw ELSE name]
        /\ used' = IF getter THEN used \cup {name, Getter(name)}
                   ELSE (used \cup {name}) \ {Getter(name)}
  /\ index' = index + 1
  /\ UNCHANGED <<order, version>>
Done == index = Count + 1
Selectors(s) == IF s \in Branches THEN {Getter(pb[s])}
                ELSE {pb[s], Getter(pb[s])}
CompilerSafe == /\ \A s \in Slots: Selectors(s) \cap ActualReserved = {}
                /\ \A s, t \in Slots: s # t => Selectors(s) \cap Selectors(t) = {}
Repair == /\ Mode = "checked" /\ Done /\ ~CompilerSafe
          /\ version' = version + 1
          /\ index' = 1 /\ used' = Reserved
          /\ pb' = [s \in Slots |-> Empty] /\ loom' = pb'
          /\ wrappers' = pb' /\ pbWrappers' = pb'
          /\ UNCHANGED order
Ready == Done /\ (Mode # "checked" \/ CompilerSafe)
Next == Step \/ Repair \/ (Ready /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
GoAgreement == \A s \in Slots: loom[s] = pb[s]
WrapperAgreement == \A s \in Slots: wrappers[s] = pbWrappers[s]
SafeOutput == Ready => CompilerSafe
SafeGetters == Ready => \A s, t \in Slots: s # t => Selectors(s) \cap Selectors(t) = {}
SafeMethods == Ready => \A s \in Slots: Selectors(s) \cap ActualReserved = {}
BoundedRepair == version <= Count
Terminates == <>Ready
=============================================================================
