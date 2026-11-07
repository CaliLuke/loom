--------------------------- MODULE FieldCodecs ---------------------------
EXTENDS Naturals
CONSTANT Policy
VARIABLES location, sourceType, leaves, nullable, media, flags, phase, captured, codec
vars == <<location, sourceType, leaves, nullable, media, flags, phase, captured, codec>>

\* Union field alternatives after schema admission. Binary whole fields are a
\* separate preserved protocol; byte alternatives inside a union use JSON.
Default(t) == CASE t = "scalar" -> "text" [] t \in {"object", "array"} -> "json" [] OTHER -> "octets"
ScalarOnly == leaves = "scalars"

\* This model starts AFTER content application has selected a complete value
\* or one array item. "array" therefore means an encoded array value, not an
\* outer named array property. SourceApplication models that prior decision.
\* Explicit flags select the supported style grammar on either media.
Expected ==
  IF flags THEN
    IF nullable THEN "reject" ELSE "text"
  ELSE LET selected == IF media = "absent" THEN Default(sourceType) ELSE media
       IN CASE selected = "json" -> "json"
            [] selected = "text" /\ ScalarOnly /\ ~nullable -> "text"
            [] OTHER -> "reject"

\* Candidate pipeline captures before source type normalization. A normalized
\* typeless union never infers media independently from each alternative.
Capture == /\ phase = "source"
           /\ captured' = IF media # "absent" THEN media
                           ELSE IF Policy = "normalize-first" THEN "octets"
                           ELSE Default(sourceType)
           /\ phase' = "normalized"
           /\ UNCHANGED <<location, sourceType, leaves, nullable, media, flags, codec>>
Resolve == /\ phase = "normalized"
           /\ codec' = IF flags THEN
                         IF nullable THEN "reject" ELSE "text"
                       ELSE IF Policy = "shape-only" /\ location = "multipart" /\ captured = "json" THEN "reject"
                       ELSE IF captured = "json" THEN "json"
                       ELSE IF captured = "text" /\ ScalarOnly /\ ~nullable /\
                               ~(Policy = "text-rejected" /\ location = "form") THEN "text"
                       ELSE "reject"
           /\ phase' = "done"
           /\ UNCHANGED <<location, sourceType, leaves, nullable, media, flags, captured>>
Init == /\ location \in {"form", "multipart"}
        /\ sourceType \in {"absent", "scalar", "object", "array"}
        /\ leaves \in {"scalars", "scalar-lists", "objects"}
        /\ nullable \in BOOLEAN /\ media \in {"absent", "text", "json", "octets"} /\ flags \in BOOLEAN
        /\ phase = "source" /\ captured = "unset" /\ codec = "unset"
Next == Capture \/ Resolve \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
Agreement == phase = "done" => codec = Expected
NoTextNull == phase = "done" /\ codec = "text" => ~nullable
NoTextComposite == phase = "done" /\ codec = "text" /\ ~flags => ScalarOnly
=============================================================================
