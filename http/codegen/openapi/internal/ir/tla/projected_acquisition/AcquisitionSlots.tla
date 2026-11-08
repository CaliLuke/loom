---------------------------- MODULE AcquisitionSlots ----------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANT Mode, Nullable, Compatible, Order
VARIABLES phase, requests, cursor, registrations, claims, selections, compareNeeded,
          cState, boundHandle, finalResult, slot, slotRead, slotReady,
          discriminator, rootRecord, nonNullRecord, mappingValues, mappingReady,
          projectionHandles, drainRef, rootRef, projectedMapping, rejected
vars == <<phase, requests, cursor, registrations, claims, selections, compareNeeded,
          cState, boundHandle, finalResult, slot, slotRead, slotReady,
          discriminator, rootRecord, nonNullRecord, mappingValues, mappingReady,
          projectionHandles, drainRef, rootRef, projectedMapping, rejected>>
EmptyResult == <<0,"">>
ExpectedResult == <<1,"authored-overlay-and-reference-local-example">>
Orders == IF Order = "Reverse" THEN {<<"preferred","canonical">>}
          ELSE {<<"preferred","canonical">>, <<"canonical","preferred">>}
Init == /\ phase = "Select" /\ requests \in Orders /\ cursor = 1
        /\ registrations = <<>> /\ claims = {} /\ selections = <<>> /\ compareNeeded = FALSE
        /\ cState = "absent" /\ boundHandle = 0 /\ finalResult = EmptyResult
        /\ slot = EmptyResult /\ slotRead = FALSE /\ slotReady = FALSE
        /\ discriminator = "" /\ rootRecord = "" /\ nonNullRecord = ""
        /\ mappingValues = [r \in {"walk-record","wrapper-record"} |-> ""]
        /\ mappingReady = FALSE /\ projectionHandles = [k \in {"exact","different-owner"} |-> ""]
        /\ drainRef = "" /\ rootRef = "" /\ projectedMapping = "" /\ rejected = FALSE
(* All registrations here have the SAME F and normalized hint. A noncanonical
   first registration has a suffixed public name; it is not a recorded claim.
   Request role is a selector, not a universal partition in the actual F bucket. *)
Select == /\ phase = "Select" /\ cursor <= Len(requests)
          /\ LET canonical == requests[cursor] = "canonical"
                 reusable == IF canonical /\ Mode # "PreferredHintOnly" THEN claims
                             ELSE (1..Len(registrations))
                 fresh == reusable = {}
                 id == IF fresh THEN Len(registrations)+1 ELSE CHOOSE x \in reusable: TRUE
             IN /\ registrations' = IF fresh THEN Append(registrations, IF canonical THEN "canonical-associated" ELSE "preferred-only-suffixed") ELSE registrations
                /\ claims' = IF fresh /\ canonical THEN claims \cup {id} ELSE claims
                /\ selections' = Append(selections,id)
                /\ compareNeeded' = (compareNeeded \/ (canonical /\ fresh /\ Len(registrations) > 0))
          /\ cursor' = cursor+1
          /\ phase' = IF cursor = Len(requests) THEN "CaptureMapping" ELSE phase
          /\ UNCHANGED <<requests,cState,boundHandle,finalResult,slot,slotRead,slotReady,discriminator,rootRecord,nonNullRecord,mappingValues,mappingReady,projectionHandles,drainRef,rootRef,projectedMapping,rejected>>
CaptureMapping == /\ phase = "CaptureMapping"
                  /\ discriminator' = "walk-record" /\ rootRecord' = "walk-record"
                  /\ phase' = "ReserveC"
                  /\ UNCHANGED <<requests,cursor,registrations,claims,selections,compareNeeded,cState,boundHandle,finalResult,slot,slotRead,slotReady,nonNullRecord,mappingValues,mappingReady,projectionHandles,drainRef,rootRef,projectedMapping,rejected>>
ReserveC == /\ phase = "ReserveC" /\ cState = "absent"
            /\ cState' = "reserved"
            /\ phase' = IF Mode = "UnboundSnapshot" THEN "Traverse" ELSE "BindC"
            /\ UNCHANGED <<requests,cursor,registrations,claims,selections,compareNeeded,boundHandle,finalResult,slot,slotRead,slotReady,discriminator,rootRecord,nonNullRecord,mappingValues,mappingReady,projectionHandles,drainRef,rootRef,projectedMapping,rejected>>
BindC == /\ phase = "BindC" /\ cState = "reserved"
         /\ boundHandle' = 1 /\ cState' = "bound" /\ phase' = "Traverse"
         /\ UNCHANGED <<requests,cursor,registrations,claims,selections,compareNeeded,finalResult,slot,slotRead,slotReady,discriminator,rootRecord,nonNullRecord,mappingValues,mappingReady,projectionHandles,drainRef,rootRef,projectedMapping,rejected>>
(* A different root/component route reenters this C during traversal.
   Candidate records a result-cell dependency. Both bad variants snapshot now. *)
Traverse == /\ phase = "Traverse"
            /\ slot' = IF Mode \in {"UnboundSnapshot","BareHandleSnapshot"} THEN <<boundHandle,"">> ELSE slot
            /\ slotRead' = (Mode \in {"UnboundSnapshot","BareHandleSnapshot"})
            /\ phase' = "WrapNullable"
            /\ UNCHANGED <<requests,cursor,registrations,claims,selections,compareNeeded,cState,boundHandle,finalResult,slotReady,discriminator,rootRecord,nonNullRecord,mappingValues,mappingReady,projectionHandles,drainRef,rootRef,projectedMapping,rejected>>
WrapNullable == /\ phase = "WrapNullable"
                /\ nonNullRecord' = IF Nullable THEN rootRecord ELSE nonNullRecord
                /\ rootRecord' = IF Nullable THEN "wrapper-record" ELSE rootRecord
                /\ phase' = "FinishResult"
                /\ UNCHANGED <<requests,cursor,registrations,claims,selections,compareNeeded,cState,boundHandle,finalResult,slot,slotRead,slotReady,discriminator,mappingValues,mappingReady,projectionHandles,drainRef,rootRef,projectedMapping,rejected>>
(* Includes post-return reference-local example/overlay completion. *)
FinishResult == /\ phase = "FinishResult"
                /\ finalResult' = ExpectedResult /\ cState' = "complete" /\ phase' = "Plan"
                /\ UNCHANGED <<requests,cursor,registrations,claims,selections,compareNeeded,boundHandle,slot,slotRead,slotReady,discriminator,rootRecord,nonNullRecord,mappingValues,mappingReady,projectionHandles,drainRef,rootRef,projectedMapping,rejected>>
Plan == /\ phase = "Plan" /\ cState = "complete"
        /\ projectionHandles' = [projectionHandles EXCEPT !["exact"] = "P1"]
        /\ drainRef' = "P1" /\ phase' = "Install"
        /\ UNCHANGED <<requests,cursor,registrations,claims,selections,compareNeeded,cState,boundHandle,finalResult,slot,slotRead,slotReady,discriminator,rootRecord,nonNullRecord,mappingValues,mappingReady,rootRef,projectedMapping,rejected>>
InstallComposition == /\ phase = "Install" /\ ~slotReady
                      /\ slot' = IF Mode = "BareHandleSnapshot" THEN slot ELSE finalResult
                      /\ slotRead' = TRUE /\ slotReady' = TRUE
                      /\ UNCHANGED <<phase,requests,cursor,registrations,claims,selections,compareNeeded,cState,boundHandle,finalResult,discriminator,rootRecord,nonNullRecord,mappingValues,mappingReady,projectionHandles,drainRef,rootRef,projectedMapping,rejected>>
InstallMapping == /\ phase = "Install" /\ ~mappingReady
                  /\ LET target == IF Mode = "OuterOwner" THEN rootRecord ELSE discriminator IN
                     mappingValues' = [mappingValues EXCEPT ![target] = drainRef]
                  /\ mappingReady' = TRUE
                  /\ UNCHANGED <<phase,requests,cursor,registrations,claims,selections,compareNeeded,cState,boundHandle,finalResult,slot,slotRead,slotReady,discriminator,rootRecord,nonNullRecord,projectionHandles,drainRef,rootRef,projectedMapping,rejected>>
Materialize == /\ phase = "Install" /\ slotReady
               /\ (mappingReady \/ Mode = "ProjectBeforeInstall")
               /\ projectedMapping' = mappingValues[IF Nullable THEN nonNullRecord ELSE rootRecord]
               /\ phase' = "Compare"
               /\ UNCHANGED <<requests,cursor,registrations,claims,selections,compareNeeded,cState,boundHandle,finalResult,slot,slotRead,slotReady,discriminator,rootRecord,nonNullRecord,mappingValues,mappingReady,projectionHandles,drainRef,rootRef,rejected>>
Compare == /\ phase = "Compare"
           /\ rejected' = (compareNeeded /\ ~Compatible) /\ phase' = IF compareNeeded /\ ~Compatible THEN "Done" ELSE "RootProjection"
           /\ UNCHANGED <<requests,cursor,registrations,claims,selections,compareNeeded,cState,boundHandle,finalResult,slot,slotRead,slotReady,discriminator,rootRecord,nonNullRecord,mappingValues,mappingReady,projectionHandles,drainRef,rootRef,projectedMapping>>
(* Scope has released; occurrence-root projection uses the same exact request
   and reservation owner as the drain. It may not manufacture a second key. *)
RootProjection == /\ phase = "RootProjection"
                  /\ rootRef' = IF Mode = "RootDifferentOwner" THEN "P2" ELSE projectionHandles["exact"]
                  /\ projectionHandles' = IF Mode = "RootDifferentOwner" THEN [projectionHandles EXCEPT !["different-owner"] = "P2"] ELSE projectionHandles
                  /\ phase' = "Done"
                  /\ UNCHANGED <<requests,cursor,registrations,claims,selections,compareNeeded,cState,boundHandle,finalResult,slot,slotRead,slotReady,discriminator,rootRecord,nonNullRecord,mappingValues,mappingReady,drainRef,projectedMapping,rejected>>
Done == /\ phase = "Done" /\ UNCHANGED vars
Next == Select \/ CaptureMapping \/ ReserveC \/ BindC \/ Traverse \/ WrapNullable \/ FinishResult \/ Plan \/ InstallComposition \/ InstallMapping \/ Materialize \/ Compare \/ RootProjection \/ Done
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
NoUnboundRead == slotRead => slot[1] # 0
BoundBeforeTraversal == phase \in {"Traverse","WrapNullable","FinishResult","Plan","Install","Compare","RootProjection","Done"} => boundHandle # 0
CompositionComplete == phase \in {"Compare","RootProjection","Done"} => slotReady /\ slot = finalResult /\ finalResult = ExpectedResult
MappingPreserved == phase \in {"Compare","RootProjection","Done"} => mappingReady /\ projectedMapping = drainRef
CanonicalAssociation == \A i \in 1..Len(selections): requests[i] = "canonical" => selections[i] \in claims
RoleOrder == phase = "Done" =>
 IF requests[1] = "canonical" THEN selections[1] = selections[2]
 ELSE selections[1] # selections[2] /\ compareNeeded
ClaimValidation == phase = "Done" => rejected = (compareNeeded /\ ~Compatible)
SameProjectionOwner == phase = "Done" /\ ~rejected => rootRef = drainRef /\ projectionHandles["different-owner"] = ""
Terminates == <>(phase = "Done")
================================================================================
