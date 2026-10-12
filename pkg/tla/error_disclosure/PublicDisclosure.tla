--------------------------- MODULE PublicDisclosure ---------------------------
EXTENDS TLC
CONSTANTS Checked, HistoryChecked
VARIABLES shape, statusMessage, detailMessage, historyMessages, original, phase
vars == <<shape, statusMessage, detailMessage, historyMessages, original, phase>>
Shapes == {"plain", "wrappedPlain", "public", "wrappedPublic", "safe",
           "joined", "reversedJoin", "safeBranch", "outerOwner", "cancel",
           "nilRemedyOuter", "codeOnlyRemedyOuter", "emptyRemedyOuter"}
HistoryShapes == {"nilRemedyOuter", "codeOnlyRemedyOuter", "emptyRemedyOuter"}
SafeOverride == shape = "safe" \/ shape \in HistoryShapes
Public == CASE shape \in {"public", "wrappedPublic", "outerOwner"} -> "public"
            [] SafeOverride -> "safe"
            [] shape = "cancel" -> "context canceled"
            [] OTHER -> "internal server error"
Legacy == CASE shape = "public" -> "public"
            [] shape = "safe" -> "safe"
            [] shape = "safeBranch" -> "safe"
            [] shape = "outerOwner" -> "public"
            [] OTHER -> "private"
Init ==
  /\ shape \in Shapes
  /\ statusMessage = "pending"
  /\ detailMessage = "pending"
  /\ historyMessages = {}
  /\ original = "retained"
  /\ phase = "encode"
Encode ==
  /\ phase = "encode"
  /\ statusMessage' = IF Checked THEN Public ELSE Legacy
  /\ detailMessage' = IF shape = "cancel" THEN "none"
                       ELSE IF Checked THEN Public ELSE Legacy
  /\ historyMessages' = IF shape \in HistoryShapes
       THEN IF HistoryChecked /\ SafeOverride THEN {"safe", "entry safe"}
            ELSE {"private", "entry safe"}
       ELSE {}
  /\ phase' = "done"
  /\ UNCHANGED <<shape, original>>
Spec == Init /\ [][Encode]_vars
PublicOwnership == phase = "done" => statusMessage = Public
NoDiagnosticDisclosure == phase = "done" =>
  statusMessage # "private" /\ detailMessage # "private" /\ "private" \notin historyMessages
SamePublicOwner == phase = "done" /\ shape # "cancel" => detailMessage = statusMessage
HistoryPublicOwner == phase = "done" /\ shape \in HistoryShapes =>
  historyMessages = {"safe", "entry safe"}
OriginalRetained == original = "retained"
NativeTermination == phase = "done" /\ shape = "cancel" => detailMessage = "none"
=============================================================================
