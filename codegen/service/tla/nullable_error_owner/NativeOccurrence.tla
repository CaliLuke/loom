--------------------------- MODULE NativeOccurrence ---------------------------
EXTENDS Naturals
CONSTANT CopyExposure
VARIABLES phase, presence, ownedName, ownedMessage, exposedName, exposedMessage,
          mutation, accepted, extractedMessage, nativeFact
vars == <<phase, presence, ownedName, ownedMessage, exposedName, exposedMessage,
          mutation, accepted, extractedMessage, nativeFact>>
Init == /\ phase = "created"
        /\ presence \in {"absent", "null", "value"}
        /\ ownedName = "declared" /\ ownedMessage = "original"
        /\ exposedName = "none" /\ exposedMessage = "none"
        /\ mutation = "none" /\ accepted = FALSE
        /\ extractedMessage = "none" /\ nativeFact = FALSE
Expose == /\ phase = "created" /\ phase' = "exposed"
          /\ exposedName' = IF presence = "value" THEN ownedName ELSE "none"
          /\ exposedMessage' = IF presence = "value" THEN ownedMessage ELSE "none"
          /\ UNCHANGED <<presence, ownedName, ownedMessage, mutation, accepted, extractedMessage, nativeFact>>
Mutate == /\ phase = "exposed" /\ phase' = "mutated"
          /\ mutation' \in {"message", "identity"}
          /\ exposedName' = IF presence = "value" /\ mutation' = "identity" THEN "other" ELSE exposedName
          /\ exposedMessage' = IF presence = "value" /\ mutation' = "message" THEN "changed" ELSE exposedMessage
          /\ ownedName' = IF CopyExposure \/ presence # "value" THEN ownedName ELSE exposedName'
          /\ ownedMessage' = IF CopyExposure \/ presence # "value" THEN ownedMessage ELSE exposedMessage'
          /\ UNCHANGED <<presence, accepted, extractedMessage, nativeFact>>
Extract == /\ phase = "mutated" /\ phase' = "done"
           /\ accepted' = (presence # "value" \/ ownedName = "declared")
           /\ nativeFact' = (presence = "value" /\ ownedName = "declared")
           /\ extractedMessage' = IF nativeFact' THEN ownedMessage ELSE "none"
           /\ UNCHANGED <<presence, ownedName, ownedMessage, exposedName, exposedMessage, mutation>>
Next == Expose \/ Mutate \/ Extract
Spec == Init /\ [][Next]_vars
MutationVisible == phase = "done" /\ presence = "value" /\ mutation = "message" => extractedMessage = "changed"
IdentityRechecked == phase = "done" /\ presence = "value" /\ mutation = "identity" => ~accepted /\ ~nativeFact
NoInventedOccurrence == nativeFact => presence = "value"
ReachedMessage == ~(phase = "done" /\ presence = "value" /\ mutation = "message")
ReachedIdentity == ~(phase = "done" /\ presence = "value" /\ mutation = "identity")
=============================================================================
