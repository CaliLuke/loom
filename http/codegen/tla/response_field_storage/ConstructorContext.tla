-------------------- MODULE ConstructorContext --------------------
EXTENDS TLC
CONSTANT Mode
VARIABLE specimen, actual, complete
vars == <<specimen, actual, complete>>
Case(name, pkg, nominal, servicePkg, serviceNominal) ==
    [name |-> name, pkg |-> pkg, nominal |-> nominal,
     servicePkg |-> servicePkg, serviceNominal |-> serviceNominal]
Cases == {
    Case("service", "service", "Label", "service", "Label"),
    Case("view", "views", "LabelView", "service", "LabelView"),
    Case("view collision", "views", "LabelView2", "service", "LabelView"),
    Case("located alias", "shared2", "Label", "shared2", "Label")
}
Init == /\ specimen \in Cases
        /\ actual = <<"", "">>
        /\ complete = FALSE
Construct == /\ ~complete
             /\ actual' = CASE Mode = "split" -> <<specimen.servicePkg, specimen.serviceNominal>>
                                [] Mode = "package" -> <<specimen.pkg, specimen.serviceNominal>>
                                [] OTHER -> <<specimen.pkg, specimen.nominal>>
             /\ complete' = TRUE
             /\ UNCHANGED specimen
Next == Construct
Spec == Init /\ [][Next]_vars
AssignmentNamesDeclaration == complete => actual = <<specimen.pkg, specimen.nominal>>
=============================================================================
