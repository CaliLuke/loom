------------------------- MODULE ResponseOwnership -------------------------
EXTENDS TLC
CONSTANT Mode
VARIABLES shape, reverse, node, owner, metadata, message, phase
vars == <<shape, reverse, node, owner, metadata, message, phase>>
Shapes == {"direct", "wrapped", "singleJoin", "joined", "wrappedJoin",
           "outerJoin", "safeWrapped", "safeJoin", "outerDifferentType"}
FirstBranch == IF reverse THEN "otherTyped" ELSE "typed"
Start == CASE shape = "direct" -> "typed"
           [] shape = "joined" -> "join"
           [] shape = "outerJoin" \/ shape = "outerDifferentType" -> "outer"
           [] shape = "safeWrapped" \/ shape = "safeJoin" -> "safe"
           [] OTHER -> "wrapper"
NextNode == CASE node = "wrapper" ->
                  IF shape = "wrappedJoin" THEN "join" ELSE "typed"
              [] node = "safe" -> IF shape = "safeJoin" THEN "join" ELSE "typed"
              [] OTHER -> "none"
ExpectedOwner == CASE shape \in {"joined", "wrappedJoin", "safeJoin"} -> "none"
                   [] shape \in {"outerJoin", "outerDifferentType"} -> "outer"
                   [] OTHER -> "typed"
ExpectedMessage == IF shape \in {"safeWrapped", "safeJoin"} THEN "safe"
                   ELSE IF ExpectedOwner = "none" THEN "generic"
                   ELSE ExpectedOwner
Init == /\ shape \in Shapes /\ reverse \in BOOLEAN
        /\ node = Start /\ owner = "pending" /\ metadata = "pending"
        /\ message = "pending" /\ phase = "walk"
Walk ==
 /\ phase = "walk"
 /\ IF node \in {"typed", "outer"} THEN
       /\ owner' = node /\ phase' = "emit" /\ UNCHANGED node
    ELSE IF node = "join" THEN
       /\ owner' = IF Mode = "legacy" THEN FirstBranch ELSE "none"
       /\ phase' = "emit" /\ UNCHANGED node
    ELSE IF Mode = "reject_wrappers" THEN
       /\ owner' = "none" /\ phase' = "emit" /\ UNCHANGED node
    ELSE /\ node' = NextNode /\ UNCHANGED <<owner, phase>>
 /\ UNCHANGED <<shape, reverse, metadata, message>>
Emit ==
 /\ phase = "emit"
 /\ metadata' = IF Mode = "legacy" /\ shape = "outerDifferentType"
                   THEN "typed" ELSE owner
 /\ message' = ExpectedMessage
 /\ phase' = "done"
 /\ UNCHANGED <<shape, reverse, node, owner>>
Spec == Init /\ [][Walk \/ Emit]_vars
CompleteOwner == phase = "done" => owner = ExpectedOwner
SameMetadataOwner == phase = "done" => metadata = owner
SafeWrapperRetainsBody == phase = "done" /\ shape = "safeWrapped" =>
                         message = "safe" /\ owner = "typed"
NoBranchPromotion == phase = "done" /\ shape \in {"joined", "wrappedJoin", "safeJoin"} =>
                    owner = "none" /\ metadata = "none"
=============================================================================
