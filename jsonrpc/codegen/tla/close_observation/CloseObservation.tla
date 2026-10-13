------------------------ MODULE CloseObservation ------------------------
EXTENDS TLC
CONSTANT Observe
VARIABLES frameSent, peerRead, nativeClosed, checked
vars == <<frameSent, peerRead, nativeClosed, checked>>
Init == /\ frameSent = FALSE /\ peerRead = FALSE /\ nativeClosed = FALSE /\ checked = FALSE
Send == /\ ~frameSent /\ frameSent' = TRUE /\ UNCHANGED <<peerRead,nativeClosed,checked>>
Read == /\ frameSent /\ ~peerRead /\ peerRead' = TRUE /\ UNCHANGED <<frameSent,nativeClosed,checked>>
Close == /\ frameSent /\ ~nativeClosed /\ nativeClosed' = TRUE /\ UNCHANGED <<frameSent,peerRead,checked>>
Check == /\ peerRead /\ ~checked
         /\ (Observe = "peer" \/ nativeClosed)
         /\ checked' = TRUE /\ UNCHANGED <<frameSent,peerRead,nativeClosed>>
Next == Send \/ Read \/ Close \/ Check
Spec == Init /\ [][Next]_vars
SoundObservation == checked => nativeClosed
=============================================================================
