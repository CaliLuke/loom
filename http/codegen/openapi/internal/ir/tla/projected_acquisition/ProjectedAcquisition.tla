------------------------ MODULE ProjectedAcquisition ------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANT Mode, Scenario
VARIABLES roots, position, stack, cache, contributors, records, completed, edges,
          preparations, annotations, annotationBindings, materialFields, outputs,
          planned, installed, snapshots, resolved, cacheReads, published, phase, cause,
          rejected, held, checkpoint
vars == <<roots, position, stack, cache, contributors, records, completed, edges,
          preparations, annotations, annotationBindings, materialFields, outputs,
          planned, installed, snapshots, resolved, cacheReads, published, phase, cause,
          rejected, held, checkpoint>>
Decls == {"Pet", "Cat", "OtherPet", "AliasA", "AliasB", "PreferredPet", "HintA", "HintB"}
Contexts == {"full", "removed"}
Nodes == Decls \X Contexts
Hint(d) == CASE d = "Cat" -> "Cat"
            [] d \in {"AliasA", "AliasB"} -> ""
            [] d \in {"HintA", "HintB"} -> d
            [] OTHER -> "Pet"
Canonical(d) == d \in {"Pet", "Cat", "OtherPet"}
Claim(d) == CASE d = "Cat" -> "Cat"
             [] d \in {"AliasA", "AliasB", "PreferredPet", "HintA", "HintB"} -> ""
             [] OTHER -> "Pet"
(* FP is a supplied existing acquisition-fingerprint abstraction, NOT logical ID.
   Aliases deliberately share an FP; OtherPet has a distinct FP and same claim. *)
FP(n) == <<CASE n[1] \in {"AliasA", "AliasB", "HintA", "HintB"} -> "shared-alias-shape"
               [] n[1] = "PreferredPet" -> "Pet"
               [] OTHER -> n[1], n[2]>>
ExactKey(n) == <<FP(n), Hint(n[1])>>
Key(n) == CASE Mode = "BroadCanonical" /\ Claim(n[1]) # "" -> <<<<Claim(n[1]), n[2]>>, Claim(n[1])>>
           [] Mode = "CollapseContext" -> <<FP(<<n[1], "full">>), Hint(n[1])>>
           [] Mode = "LogicalKey" -> <<FP(n), Hint(n[1]), n[1]>>
           [] Mode = "CanonicalPartition" -> <<FP(n), Hint(n[1]), Canonical(n[1])>>
           [] OTHER -> ExactKey(n)
Keys == {Key(n): n \in Nodes}
(* Independent routing key; declaration remains appropriate here. *)
ContributorKey(n) == <<n[1], n[2]>>
ContributorKeys == {ContributorKey(n): n \in Nodes}
Dependency(n) == IF n[1] \in {"Pet", "OtherPet"} THEN <<"Cat", n[2]>> ELSE <<"Pet", n[2]>>
HasDependency(n) == n[1] \in {"Pet", "Cat"} \/ (Scenario = "EqualClaims" /\ n[1] = "OtherPet")
ExpectedFields(n) == IF n[1] = "Cat" /\ n[2] = "full" THEN {"kind", "thumb"} ELSE {"kind"}
BuiltFields(n) == IF Mode = "WrongFields" /\ n[1] = "Cat" THEN {"kind"} ELSE ExpectedFields(n)
Authorities == {"Pet", "Cat", "OtherPet", "alias", "HintA", "HintB"} \X {"baseline", "prepared-other"}
Authority(n) == <<IF n[1] \in {"AliasA", "AliasB"} THEN "alias" ELSE IF n[1] = "PreferredPet" THEN "Pet" ELSE n[1],
                  IF Scenario = "AnnotationOrigins" /\ n[2] = "removed" THEN "prepared-other" ELSE "baseline">>
BoundAuthority(n) == IF Mode = "WrongAnnotation" THEN <<Authority(n)[1], "baseline">> ELSE Authority(n)
(* Supplied compatibility facts, independent of Key, record allocation and mode. *)
Equivalent(a,b) == a[1] = b[1] \/ {a[1],b[1]} \subseteq {"AliasA","AliasB"}
                   \/ (Scenario = "EqualClaims" /\ {a[1],b[1]} = {"Pet","OtherPet"})
Conflicts == \E i,j \in 1..Len(records):
 Claim(records[i][1]) # "" /\ Claim(records[i][1]) = Claim(records[j][1])
 /\ records[i][1] # records[j][1] /\ ~Equivalent(records[i],records[j])
RECURSIVE Reach(_, _)
Reach(ids, steps) == IF steps = 0 THEN ids ELSE
 Reach(ids \cup {e[2]: e \in {x \in edges: x[1] \in ids}}, steps-1)
Closure(id) == Reach({id}, Len(records))
NormalOrders == {<<<<"Pet","full">>, <<"Pet","removed">>>>,
                 <<<<"Pet","removed">>, <<"Pet","full">>>>,
                 <<<<"Cat","full">>, <<"Cat","removed">>>>,
                 <<<<"Cat","removed">>, <<"Cat","full">>>>,
                 <<<<"Pet","full">>, <<"Cat","full">>, <<"Pet","full">>>>}
RootOrders == CASE Scenario \in {"Conflict", "EqualClaims"} -> {<<<<"Pet","full">>, <<"OtherPet","full">>>>}
               [] Scenario = "FailureConflict" -> {<<<<"OtherPet","full">>, <<"Pet","full">>>>}
               [] Scenario = "StructuralAliases" -> {<<<<"AliasA","full">>, <<"AliasB","full">>>>}
               [] Scenario = "CrossRole" -> {<<<<"Pet","full">>, <<"PreferredPet","full">>>>}
               [] Scenario = "DifferentHints" -> {<<<<"HintA","full">>, <<"HintB","full">>>>}
               [] Scenario = "LookupReuse" -> {<<<<"OtherPet","full">>, <<"OtherPet","full">>>>}
               [] OTHER -> NormalOrders
Init == /\ roots \in RootOrders /\ position = 1 /\ stack = <<>>
        /\ cache = [k \in Keys |-> 0] /\ contributors = [k \in ContributorKeys |-> 0]
        /\ records = <<>> /\ completed = {} /\ edges = {}
        /\ preparations = [k \in ContributorKeys |-> 0]
        /\ annotations = [b \in Authorities |-> 0] /\ annotationBindings = <<>>
        /\ materialFields = <<>> /\ outputs = <<>>
        /\ planned = {} /\ installed = {} /\ snapshots = {} /\ resolved = {} /\ cacheReads = {}
        /\ published = {} /\ phase = "Walk" /\ cause = "none" /\ rejected = FALSE
        /\ held = FALSE /\ checkpoint = 1
Acquire(n, parent, root) ==
 LET ck == ContributorKey(n)
     routed == IF root THEN 0 ELSE contributors[ck]
     prior == cache[Key(n)]
     (* Current canonical lookup rejects a suffixed provisional public name.
        LateContributor alone retains correct component lookup. *)
     matching == prior # 0 /\ (Mode \notin {"Current", "PublicFilter"} \/ ~Canonical(n[1]))
     reuse == routed # 0 \/ matching
     id == IF routed # 0 THEN routed ELSE IF matching THEN prior ELSE Len(records)+1
     nextStack == IF root THEN stack ELSE [stack EXCEPT ![Len(stack)].step = 1]
     prepare == ~root /\ routed = 0
     binding == BoundAuthority(n)
 IN /\ phase = "Walk"
    /\ records' = IF reuse THEN records ELSE Append(records, n)
    /\ cache' = IF reuse THEN cache ELSE [cache EXCEPT ![Key(n)] = id]
    /\ contributors' = IF root \/ Mode \in {"Current", "LateContributor"} THEN contributors
                        ELSE [contributors EXCEPT ![ck] = id]
    /\ preparations' = IF prepare THEN [preparations EXCEPT ![ck] = @+1] ELSE preparations
    /\ annotations' = IF reuse \/ (annotations[binding] > 0 /\ Mode # "FreshAnnotation") THEN annotations
                       ELSE [annotations EXCEPT ![binding] = @+1]
    /\ annotationBindings' = IF reuse THEN annotationBindings ELSE Append(annotationBindings, binding)
    /\ materialFields' = IF reuse THEN materialFields ELSE Append(materialFields, BuiltFields(n))
    /\ stack' = IF reuse THEN nextStack ELSE Append(nextStack, [id |-> id, step |-> 0])
    /\ edges' = IF root \/ (Mode = "DropBackEdge" /\ reuse /\ id \notin completed)
                 THEN edges ELSE edges \cup {<<parent,id>>}
    /\ outputs' = IF root THEN Append(outputs,id) ELSE outputs
    /\ position' = IF root THEN position+1 ELSE position
    /\ held' = TRUE /\ checkpoint' = IF root THEN Len(records)+1 ELSE checkpoint
    /\ resolved' = IF Mode = "EarlyProjection" THEN resolved \cup {id} ELSE resolved
    /\ published' = IF Mode = "EarlyPublish" THEN published \cup {id} ELSE published
    /\ UNCHANGED <<roots,completed,planned,installed,snapshots,cacheReads,phase,cause,rejected>>
Root == /\ ~held /\ stack = <<>> /\ position <= Len(roots) /\ Acquire(roots[position],0,TRUE)
Child == /\ Len(stack) > 0
         /\ LET f == stack[Len(stack)] IN
            /\ f.step = 0 /\ HasDependency(records[f.id]) /\ Acquire(Dependency(records[f.id]),f.id,FALSE)
Leaf == /\ phase = "Walk" /\ Len(stack) > 0
        /\ LET f == stack[Len(stack)] IN /\ f.step = 0 /\ ~HasDependency(records[f.id])
        /\ stack' = [stack EXCEPT ![Len(stack)].step = 1]
        /\ UNCHANGED <<roots,position,cache,contributors,records,completed,edges,preparations,annotations,annotationBindings,materialFields,outputs,planned,installed,snapshots,resolved,cacheReads,published,phase,cause,rejected,held,checkpoint>>
Finish == /\ phase = "Walk" /\ Len(stack) > 0 /\ stack[Len(stack)].step = 1
          /\ LET id == stack[Len(stack)].id
                 failure == Scenario \in {"Failure","FailureConflict"} /\ records[id][1] = "Cat"
             IN /\ completed' = IF failure THEN completed ELSE completed \cup {id}
                /\ cause' = IF failure /\ cause = "none" THEN "body-error" ELSE cause
                /\ phase' = IF failure THEN "Rollback" ELSE phase
                /\ contributors' = IF Mode \in {"Current","LateContributor"} /\ ~failure
                                      THEN [contributors EXCEPT ![ContributorKey(records[id])] = id] ELSE contributors
                /\ resolved' = IF Mode = "ProjectOnFinish" /\ ~failure THEN resolved \cup {id} ELSE resolved
                /\ cacheReads' = IF Mode = "CacheOnFinish" /\ ~failure THEN cacheReads \cup {id} ELSE cacheReads
          /\ stack' = SubSeq(stack,1,Len(stack)-1)
          /\ UNCHANGED <<roots,position,cache,records,edges,preparations,annotations,annotationBindings,materialFields,outputs,planned,installed,snapshots,published,rejected,held,checkpoint>>
BeginDrain == /\ phase = "Walk" /\ held /\ stack = <<>>
              /\ phase' = IF Mode = "CompareBeforeDrain" THEN "Compare" ELSE "Plan"
              /\ UNCHANGED <<roots,position,stack,cache,contributors,records,completed,edges,preparations,annotations,annotationBindings,materialFields,outputs,planned,installed,snapshots,resolved,cacheReads,published,cause,rejected,held,checkpoint>>
Plan == /\ phase = "Plan" /\ planned # completed
        /\ \E id \in completed \ planned:
           /\ Closure(id) \subseteq completed /\ planned' = planned \cup {id}
        /\ UNCHANGED <<roots,position,stack,cache,contributors,records,completed,edges,preparations,annotations,annotationBindings,materialFields,outputs,installed,snapshots,resolved,cacheReads,published,phase,cause,rejected,held,checkpoint>>
BeginInstall == /\ phase = "Plan" /\ planned = completed /\ phase' = "Install"
                /\ UNCHANGED <<roots,position,stack,cache,contributors,records,completed,edges,preparations,annotations,annotationBindings,materialFields,outputs,planned,installed,snapshots,resolved,cacheReads,published,cause,rejected,held,checkpoint>>
Install == /\ phase = "Install" /\ installed # completed
           /\ \E id \in completed \ installed:
              /\ installed' = installed \cup {id}
           /\ UNCHANGED <<roots,position,stack,cache,contributors,records,completed,edges,preparations,annotations,annotationBindings,materialFields,outputs,planned,snapshots,resolved,cacheReads,published,phase,cause,rejected,held,checkpoint>>
BeginProject == /\ phase = "Install" /\ installed = completed /\ phase' = "Project"
                /\ UNCHANGED <<roots,position,stack,cache,contributors,records,completed,edges,preparations,annotations,annotationBindings,materialFields,outputs,planned,installed,snapshots,resolved,cacheReads,published,cause,rejected,held,checkpoint>>
Project == /\ phase = "Project" /\ resolved # completed
           /\ \E id \in completed \ resolved:
              /\ Closure(id) \subseteq completed /\ Closure(id) \subseteq installed
              /\ resolved' = resolved \cup {id} /\ cacheReads' = cacheReads \cup {id}
              /\ snapshots' = IF Mode = "StaleSnapshot" THEN snapshots ELSE snapshots \cup {id}
           /\ UNCHANGED <<roots,position,stack,cache,contributors,records,completed,edges,preparations,annotations,annotationBindings,materialFields,outputs,planned,installed,published,phase,cause,rejected,held,checkpoint>>
BeginCompare == /\ phase = "Project" /\ resolved = completed /\ phase' = "Compare"
                /\ UNCHANGED <<roots,position,stack,cache,contributors,records,completed,edges,preparations,annotations,annotationBindings,materialFields,outputs,planned,installed,snapshots,resolved,cacheReads,published,cause,rejected,held,checkpoint>>
Compare == /\ phase = "Compare"
           /\ LET conflict == Conflicts /\ Mode # "IgnoreConflict" IN
              /\ phase' = IF conflict THEN "Rollback" ELSE IF position > Len(roots) THEN "Done" ELSE "Walk"
              /\ cause' = IF conflict /\ cause = "none" THEN "canonical-conflict" ELSE cause
              /\ held' = conflict
              /\ published' = IF ~conflict /\ position > Len(roots) THEN (1..Len(records)) ELSE published
           /\ UNCHANGED <<roots,position,stack,cache,contributors,records,completed,edges,preparations,annotations,annotationBindings,materialFields,outputs,planned,installed,snapshots,resolved,cacheReads,rejected,checkpoint>>
Rollback == /\ phase = "Rollback"
            /\ cache' = [k \in Keys |-> IF Mode = "RetainFailed" \/ cache[k] < checkpoint THEN cache[k] ELSE 0]
            /\ contributors' = [k \in ContributorKeys |-> IF Mode = "RetainFailed" \/ contributors[k] < checkpoint THEN contributors[k] ELSE 0]
            /\ cause' = IF Mode = "OverwriteCause" /\ Conflicts THEN "canonical-conflict" ELSE cause
            /\ stack' = <<>> /\ held' = FALSE /\ rejected' = TRUE /\ phase' = "Done"
            /\ published' = IF Mode = "PublishFailure" THEN (1..Len(records)) ELSE {}
            /\ UNCHANGED <<roots,position,records,completed,edges,preparations,annotations,annotationBindings,materialFields,outputs,planned,installed,snapshots,resolved,cacheReads,checkpoint>>
Done == /\ phase = "Done" /\ UNCHANGED vars
Next == Root \/ Child \/ Leaf \/ Finish \/ BeginDrain \/ Plan \/ BeginInstall \/ Install \/ BeginProject \/ Project \/ BeginCompare \/ Compare \/ Rollback \/ Done
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
TypeOK == /\ records \in Seq(Nodes) /\ completed \subseteq 1..Len(records)
          /\ published \subseteq 1..Len(records) /\ edges \subseteq (1..Len(records)) \X (1..Len(records))
          /\ cause \in {"none","body-error","canonical-conflict"}
UniqueReservation == \A i,j \in 1..Len(records): ExactKey(records[i]) = ExactKey(records[j]) => i = j
OwnContext == \A i \in 1..Len(outputs): ExactKey(records[outputs[i]]) = ExactKey(roots[i])
StructuralDedup == Scenario = "StructuralAliases" /\ phase = "Done" => outputs[1] = outputs[2]
CrossRoleReuse == Scenario = "CrossRole" /\ phase = "Done" => outputs[1] = outputs[2]
DifferentHintsDistinct == Scenario = "DifferentHints" /\ phase = "Done" => outputs[1] # outputs[2]
OnePreparation == \A k \in ContributorKeys: preparations[k] <= 1
OneAnnotationAcquisition == \A b \in Authorities: annotations[b] <= 1
OwnAnnotationBinding == \A i \in 1..Len(records): annotationBindings[i] = Authority(records[i])
NoIncompleteProjection == \A id \in resolved: Closure(id) \subseteq completed
CachesAfterSlots == \A id \in cacheReads: Closure(id) \subseteq installed /\ Closure(id) \subseteq completed
ComparisonAfterSlots == phase = "Compare" => installed = completed /\ snapshots = completed /\ resolved = completed
HeldDrain == phase \in {"Plan","Install","Project","Compare"} => held
ReservationBeforeSlots == installed \subseteq planned
NoIncompletePublication == published \subseteq completed /\ published \subseteq resolved /\ published \subseteq installed
AllBackEdgesRetained == phase = "Done" /\ ~rejected =>
 \A i \in published: HasDependency(records[i]) => \E j \in published:
 <<i,j>> \in edges /\ ExactKey(records[j]) = ExactKey(Dependency(records[i]))
OwnClosure == phase = "Done" /\ ~rejected =>
 \A r \in 1..Len(outputs): HasDependency(roots[r]) =>
 \E c \in Closure(outputs[r]): records[c][1] = "Cat" /\ materialFields[c] = ExpectedFields(<<"Cat",roots[r][2]>>)
CanonicalConflict == phase = "Done" /\ Conflicts => rejected /\ published = {}
EqualClaimSuccess == Scenario = "EqualClaims" /\ phase = "Done" => ~rejected /\ published # {}
RollbackIndices == phase = "Done" /\ rejected =>
 (\A k \in Keys: cache[k] < checkpoint) /\ (\A k \in ContributorKeys: contributors[k] < checkpoint)
FailurePreserved == phase = "Done" /\ Scenario \in {"Failure","FailureConflict"} => cause = "body-error" /\ rejected /\ published = {}
Terminates == <>(phase = "Done")
=============================================================================
