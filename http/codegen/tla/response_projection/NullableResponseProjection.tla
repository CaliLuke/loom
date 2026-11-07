----------------------- MODULE NullableResponseProjection -----------------------
EXTENDS Naturals
CONSTANT Policy
VARIABLES direction, mode, parent, child, header, required, childNullable,
          nullableParent, viewed, phase, accessesFields, concreteReceiver,
          nativeSource, accepted, output
vars == <<direction, mode, parent, child, header, required, childNullable,
          nullableParent, viewed, phase, accessesFields, concreteReceiver,
          nativeSource, accepted, output>>
Presence == {"absent", "null", "value"}
Init == /\ direction \in {"encode", "decode"}
        /\ mode \in {"selected", "object", "whole", "metadata"}
        /\ nullableParent \in BOOLEAN
        /\ (mode = "object" => nullableParent)
        /\ parent \in IF nullableParent THEN Presence ELSE {"absent", "value"}
        /\ child \in Presence
        /\ header \in BOOLEAN
        /\ required \in BOOLEAN
        /\ childNullable \in BOOLEAN
        /\ viewed \in BOOLEAN
        /\ phase = "ready"
        /\ accessesFields = FALSE
        /\ concreteReceiver = FALSE
        /\ nativeSource = FALSE
        /\ accepted = FALSE
        /\ output = "none"
ChildValid == child = "value" \/ (child = "null" /\ childNullable)
BindingsValid == ~required \/ header
ParentAvailable == direction = "decode" \/ parent = "value"
SelectedValid == ChildValid /\ BindingsValid /\ ParentAvailable
WholeValid == parent # "absent" /\ BindingsValid /\
              (parent = "value" \/ ~header)
MetadataValid == BindingsValid /\ ParentAvailable
Valid == CASE mode = "selected" -> SelectedValid
           [] mode \in {"metadata", "object"} -> MetadataValid
           [] OTHER -> WholeValid
Expected == CASE mode = "selected" -> IF direction = "encode" THEN child ELSE "value"
              [] mode \in {"metadata", "object"} -> "value"
              [] OTHER -> parent
ParentOnlyAccepts == mode = "selected" /\ direction = "encode" /\
                     ParentAvailable /\ BindingsValid /\ child = "absent"
Lower == /\ phase = "ready"
         /\ phase' = "done"
         /\ accessesFields' = (Policy = "wrapper-fields" \/
                              (Valid /\ (mode # "whole" \/ parent = "value")))
         /\ concreteReceiver' = (Policy # "wrapper-fields" /\
                                  ~(Policy = "missing-object" /\ mode = "object" /\ nullableParent))
         /\ nativeSource' = ~(Policy = "view-wrapper" /\ viewed /\ direction = "encode")
         /\ accepted' = IF Policy = "whole-body-reuse" /\ mode = "metadata" /\ direction = "decode"
                        THEN FALSE
                        ELSE (Valid \/
                              (Policy = "parent-only" /\ ParentOnlyAccepts) \/
                              (Policy = "zero-parent" /\ mode = "selected" /\ direction = "encode") \/
                              (Policy = "drop-header" /\ mode = "whole" /\ parent = "null" /\ ~required))
         /\ output' = IF Policy = "zero-parent" /\ mode = "selected" /\ parent # "value"
                      THEN "value" ELSE Expected
         /\ UNCHANGED <<direction, mode, parent, child, header, required, childNullable,
                         nullableParent, viewed>>
Spec == Init /\ [][Lower]_vars
FieldStorage == accessesFields => concreteReceiver
NativeBinding == accessesFields => nativeSource
CheckedPublication == accepted => Valid
ValidReconstruction == phase = "done" /\ Valid => accepted
SelectedParent == accepted /\ mode \in {"selected", "object", "metadata"} /\ direction = "encode" => parent = "value"
PresencePreserved == accepted => output = Expected
=============================================================================
