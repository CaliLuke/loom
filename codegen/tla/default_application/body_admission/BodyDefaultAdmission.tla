-------------------- MODULE BodyDefaultAdmission --------------------
EXTENDS Naturals, TLC
CONSTANT Mode
VARIABLES input, phase, value, admitted, applied, output
vars == <<input, phase, value, admitted, applied, output>>
Absent == "absent-field"
AbsentBody == "absent-body"
Rejected == "rejected"
Pending == "pending"
Base == [body |-> TRUE, bodyRequired |-> TRUE, required |-> FALSE,
         nullable |-> FALSE, selection |-> "ordinary", raw |-> Absent,
         bodyDefault |-> "body9", role |-> "body"]
Cases == {
 [Base EXCEPT !.body = FALSE, !.bodyRequired = FALSE],
 [Base EXCEPT !.body = FALSE],
 [Base EXCEPT !.required = TRUE],
 Base,
 [Base EXCEPT !.bodyDefault = "none"],
 [Base EXCEPT !.raw = "zero"],
 [Base EXCEPT !.raw = "empty-collection"],
 [Base EXCEPT !.raw = "nonnull-value"],
 [Base EXCEPT !.raw = "null", !.nullable = TRUE],
 [Base EXCEPT !.raw = "null"],
 [Base EXCEPT !.selection = "unmatched"],
 [Base EXCEPT !.selection = "ambiguous"],
 [Base EXCEPT !.selection = "selected"],
 [Base EXCEPT !.role = "generic"]}
RawAccepted(i) == i.body /\ i.selection \in {"ordinary", "selected"} /\
                 (~i.required \/ i.raw # Absent) /\
                 (i.raw # "null" \/ i.nullable)
Expected(i) ==
 IF ~i.body THEN IF i.bodyRequired THEN Rejected ELSE AbsentBody
 ELSE IF ~RawAccepted(i) THEN Rejected
 ELSE IF i.raw # Absent THEN i.raw
 ELSE IF i.role = "body" /\ i.bodyDefault # "none" THEN i.bodyDefault
 ELSE "payload7"
\* The generic conversion remains target-owned in every mode.
TargetConvert(v) == IF v = Absent THEN "payload7" ELSE v

Init == /\ input \in Cases
        /\ phase = "raw"
        /\ value = input.raw
        /\ admitted = FALSE
        /\ applied = FALSE
        /\ output = Pending

Begin ==
 /\ phase = "raw"
 /\ LET early == Mode = "before-admission" /\ input.role = "body" /\
                   value = Absent /\ input.bodyDefault # "none"
    IN /\ value' = IF early THEN input.bodyDefault ELSE value
       /\ applied' = early
 /\ phase' = "admit"
 /\ UNCHANGED <<input, admitted, output>>

Admit ==
 /\ phase = "admit"
 /\ LET bodyPresent == input.body \/ (Mode = "invent-body" /\ ~input.bodyRequired)
        selected == input.selection \in {"ordinary", "selected"} \/ Mode = "guess-union"
        valid == bodyPresent /\ selected /\ (~input.required \/ value # Absent) /\
                 (value # "null" \/ input.nullable)
    IN /\ admitted' = valid
       /\ phase' = IF valid THEN "body-default" ELSE "done"
       /\ output' = IF valid THEN Pending ELSE
                      IF ~bodyPresent /\ ~input.bodyRequired THEN AbsentBody ELSE Rejected
 /\ UNCHANGED <<input, value, applied>>

Materialize ==
 /\ phase = "body-default"
 /\ LET earlyTarget == IF Mode = "target-first" THEN TargetConvert(value) ELSE value
        missing == earlyTarget = Absent \/
                   (Mode = "replace-supplied" /\ earlyTarget \in {"zero", "empty-collection"}) \/
                   (Mode = "replace-null" /\ earlyTarget = "null")
        doApply == Mode # "current" /\
                   (input.role = "body" \/ Mode = "global-source-default") /\
                   input.bodyDefault # "none" /\ missing
    IN /\ value' = IF doApply THEN input.bodyDefault ELSE earlyTarget
       /\ applied' = (applied \/ doApply)
 /\ phase' = "convert"
 /\ UNCHANGED <<input, admitted, output>>

Convert == /\ phase = "convert"
           /\ output' = TargetConvert(value)
           /\ phase' = "done"
           /\ UNCHANGED <<input, value, admitted, applied>>
Next == Begin \/ Admit \/ Materialize \/ Convert
Spec == Init /\ [][Next]_vars
AdmissionBeforeDefaults == applied => admitted
RejectUnadmitted == phase = "done" /\ input.body /\ ~RawAccepted(input) => output = Rejected
PreserveWholeAbsence == phase = "done" /\ ~input.body /\ ~input.bodyRequired => output = AbsentBody
PreserveSupplied == phase = "done" /\ RawAccepted(input) /\ input.raw # Absent => output = input.raw
BodyDefaultWins == phase = "done" /\ RawAccepted(input) /\ input.role = "body" /\
                   input.raw = Absent /\ input.bodyDefault # "none" => output = input.bodyDefault
GenericTargetDefault == phase = "done" /\ RawAccepted(input) /\ input.raw = Absent /\
                       (input.role = "generic" \/ input.bodyDefault = "none") => output = "payload7"
OwnOutcome == phase = "done" => output = Expected(input)
=====================================================================
