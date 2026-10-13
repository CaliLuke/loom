----------------------- MODULE SelectedViewCarrier -----------------------
EXTENDS TLC
CONSTANT Mode
VARIABLES carrier, role, state, selected, required, phase, operation, argument, outcome, entryOperation, envelope, priorCarrier, payloadCarrier, conversionArgument, roundTrip
vars == <<carrier, role, state, selected, required, phase, operation, argument, outcome, entryOperation, envelope, priorCarrier, payloadCarrier, conversionArgument, roundTrip>>
Carriers == {"pointer", "occurrence", "intrinsic"}
States(c) == IF c = "pointer" THEN {"absent", "valid", "invalid"}
             ELSE {"absent", "null", "valid", "invalid"}
Init == /\ carrier \in Carriers
        /\ role \in {"field", "element", "root"}
        /\ state \in States(carrier)
        /\ selected \in BOOLEAN
        /\ required \in BOOLEAN
        \* Existing ordinary root admission excludes nil before view validation.
        /\ ~(role = "root" /\ carrier = "pointer" /\ state = "absent")
        /\ (role = "root" => selected)
        /\ phase = "before"
        /\ operation = "none"
        /\ argument = "none"
        /\ outcome = "pending"
        /\ entryOperation = "none"
        /\ envelope = "none"
        /\ priorCarrier \in {"pointer", "occurrence"}
        /\ payloadCarrier = "none"
        /\ conversionArgument = "none"
        /\ roundTrip = "none"
MustExist == required \/ role \in {"element", "root"}
Concrete == state \in {"valid", "invalid"}
Expected == IF ~selected THEN "skip"
            ELSE IF state = "absent" THEN IF MustExist THEN "missing" ELSE "skip"
            ELSE IF state = "null" THEN "skip"
            ELSE IF state = "invalid" THEN "invalid" ELSE "valid"
Op == IF ~selected THEN "none"
      ELSE IF Mode = "old" /\ carrier # "pointer"
           THEN IF role = "root" THEN "wrapper-selector" ELSE "nil-wrapper"
      ELSE IF carrier = "pointer" THEN "nil-pointer" ELSE "presence"
Arg == IF ~selected \/ ~Concrete THEN "none"
       ELSE IF carrier = "pointer" THEN "pointer"
       ELSE IF Mode = "guard-only" THEN "wrapper-to-pointer"
       ELSE IF carrier = "intrinsic" /\ role # "root" THEN "wrapper-to-wrapper"
       ELSE "address-value"
Result == IF Mode = "null-as-missing" /\ selected /\ state = "null" /\ MustExist
          THEN "missing"
          ELSE IF Mode = "unwrap-only" /\ selected /\ state = "absent"
          THEN "skip"
          ELSE Expected
EntryOp == IF role # "root" THEN "none"
           ELSE IF carrier = "pointer" THEN "nil-pointer"
           ELSE IF Mode \in {"old", "projection-nil"} THEN "nil-wrapper"
           ELSE "presence"
Envelope == IF role # "root" THEN "none"
            ELSE IF carrier = "intrinsic" /\ Mode \in {"old", "envelope-wrap"}
                 THEN "nullable-envelope" ELSE "plain-envelope"
PayloadCarrier == IF role # "root" THEN "none"
                  ELSE IF carrier = "occurrence" /\ Mode = "occurrence-drop" THEN "pointer"
                  ELSE IF carrier # "intrinsic" /\ Mode = "envelope-reuse" THEN priorCarrier
                  ELSE carrier
ConversionArgument == IF role # "root" \/ ~Concrete THEN "none"
                      ELSE IF carrier = "intrinsic" THEN "wrapper"
                      ELSE IF carrier = "occurrence" /\ Mode = "occurrence-call" THEN "wrapper"
                      ELSE "pointer"
RoundTrip == IF role # "root" THEN "none"
             ELSE IF state = "absent" THEN "rejected"
             ELSE IF carrier = "occurrence" /\ state = "null" /\ Mode = "null-erasure"
                  THEN "absent" ELSE state
Validate == /\ phase = "before"
            /\ phase' = "after"
            /\ operation' = Op
            /\ argument' = Arg
            /\ outcome' = Result
            /\ entryOperation' = EntryOp
            /\ envelope' = Envelope
            /\ payloadCarrier' = PayloadCarrier
            /\ conversionArgument' = ConversionArgument
            /\ roundTrip' = RoundTrip
            /\ UNCHANGED <<carrier, role, state, selected, required, priorCarrier>>
Spec == Init /\ [][Validate]_vars
LegalOperations == operation \notin {"nil-wrapper", "wrapper-selector"}
ProjectionEntrySafe == entryOperation # "nil-wrapper"
EnvelopeIndependent == envelope # "nullable-envelope"
CallAgreement == argument # "wrapper-to-pointer"
AdmissionAndValidation == phase = "after" => outcome = Expected
ExcludedUntouched == phase = "after" /\ ~selected => /\ operation = "none"
                                                   /\ argument = "none"
                                                   /\ outcome = "skip"
OccurrencePreserved == phase = "after" /\ role = "root" => payloadCarrier = carrier
ConversionCallAgreement == phase = "after" /\ role = "root" /\ Concrete =>
                           conversionArgument = IF carrier = "intrinsic" THEN "wrapper" ELSE "pointer"
RoundTripPreserved == phase = "after" /\ role = "root" =>
                     roundTrip = IF state = "absent" THEN "rejected" ELSE state
=============================================================================
