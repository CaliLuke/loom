import ValueContract.Resolve

namespace ValueContract.Candidate.ResolutionControls

def keys : KeyCodec where
  encodeInteger value := toString value
  encodeDecimal _ _ := "0"
  schemaNumber _ := none
  decodeInteger _ := none
  decodeDecimal _ := none

def checks : ExternalScalarChecks := fun _ _ => true

def sid : Identity := ⟨1, 1⟩
def iid : Identity := ⟨1, 2⟩
def aid : Identity := ⟨1, 3⟩
def oid : Identity := ⟨1, 4⟩
def uid : Identity := ⟨1, 5⟩
def bid : Identity := ⟨1, 6⟩
def fieldA : Identity := ⟨2, 1⟩
def fieldB : Identity := ⟨2, 2⟩
def branchA : Identity := ⟨3, 1⟩
def branchB : Identity := ⟨3, 2⟩

def scalars : Declarations := [
  ⟨sid, 0, .scalar .string {}, none⟩,
  ⟨iid, 0, .scalar .integer {}, none⟩,
  ⟨aid, 0, .any, none⟩,
  ⟨bid, 0, .scalar .bytes {}, none⟩]

def aliases : Declarations := scalars ++ [
  ⟨oid, 0, .object [⟨fieldA, "a", "b", true, sid⟩,
    ⟨fieldB, "b", "c", true, iid⟩] false, none⟩]

/-- A legal cross-overlap still validates every retained assignment. The legacy
Go adapter instead copies an integer into both the string and integer fields. -/
theorem aliasOverlapRejects :
    resolve aliases checks keys .authoredExample oid
      (.object [("b", .scalar (.integer 1))]) = .error .invalid := by cbv

def precedence : Declarations := scalars ++ [
  ⟨oid, 0, .object [⟨fieldA, "source", "wire", true, sid⟩] false, none⟩]

theorem invalidLosingAliasRejects :
    resolve precedence checks keys .authoredExample oid
      (.object [("source", .scalar (.integer 1)), ("wire", .scalar (.string "ok"))]) =
      .error .invalid := by cbv

theorem validWireAliasWins :
    resolve precedence checks keys .authoredExample oid
      (.object [("source", .scalar (.string "first")), ("wire", .scalar (.string "second"))]) =
      .ok ⟨.object [(fieldA, .scalar (.string "second"))] [], []⟩ := by cbv

theorem duplicateBeforePrecedence :
    resolve precedence checks keys .authoredExample oid
      (.object [("wire", .scalar (.string "first")), ("wire", .scalar (.string "second"))]) =
      .error .invalid := by cbv

theorem incompleteExampleRetained :
    resolve precedence checks keys .authoredExample oid (.object []) =
      .ok ⟨.object [(fieldA, .absent)] [], [[fieldA]]⟩ := by cbv

theorem incompleteDefaultRejected :
    resolve precedence checks keys .defaultValue oid (.object []) = .error .invalid := by cbv

def unionGraph : Declarations := precedence ++ [
  ⟨uid, 1, .union uid [⟨branchA, "record", oid⟩, ⟨branchB, "any", aid⟩], none⟩]

/-- Non-object preference includes Any, preserving complete-set ambiguity. -/
theorem anyCompleteAmbiguity :
    resolve unionGraph checks keys .authoredExample uid
      (.object [("wire", .scalar (.string "ok"))]) = .error .ambiguous := by cbv

theorem completeAnyBeatsPartialObject :
    resolve unionGraph checks keys .authoredExample uid (.object []) =
      .ok ⟨.union uid branchB (.any (.object [] [])), []⟩ := by cbv

theorem selectedBranchRetained :
    resolve unionGraph checks keys .authoredExample uid
      (.selected uid branchA (.object [("wire", .scalar (.string "ok"))])) =
      .ok ⟨.union uid branchA (.object [(fieldA, .scalar (.string "ok"))] []), []⟩ := by cbv

theorem wrongOccurrenceRejected :
    resolve unionGraph checks keys .authoredExample uid
      (.selected oid branchA (.object [])) = .error .invalid := by cbv

theorem anyBytesStayBytes :
    resolve scalars checks keys .authoredExample aid (.scalar (.bytes [104, 105])) =
      .ok ⟨.any (.scalar (.bytes [104, 105])), []⟩ := by cbv

theorem anyNilBytesStayNil :
    resolve scalars checks keys .authoredExample aid .nilBytes =
      .ok ⟨.any .nilBytes, []⟩ := by cbv

theorem declaredNilBytesNormalize :
    resolve scalars checks keys .authoredExample bid .nilBytes =
      .ok ⟨.scalar (.bytes []), []⟩ := by cbv

theorem declaredBytesTextCoerces :
    resolve scalars checks keys .authoredExample bid (.scalar (.string "hi")) =
      .ok ⟨.scalar (.bytes [104, 105]), []⟩ := by cbv

theorem anyKeyCollisionRejected :
    resolve scalars checks keys .authoredExample aid
      (.map [(.integer 1, .null), (.string "1", .null)]) = .error .invalid := by cbv

def recursive : Declarations := [⟨oid, 0, .array oid {}, none⟩]

theorem recursiveFiniteAccepted :
    resolve recursive checks keys .authoredExample oid
      (.array [.array [.array []]]) = .ok ⟨.array [.array [.array []]], []⟩ := by cbv

theorem explicitCycleRejected :
    resolve recursive checks keys .authoredExample oid (.array [.cycle 42]) =
      .error .cyclic := by cbv

def nestedID : Identity := ⟨4, 1⟩
def outerID : Identity := ⟨4, 2⟩
def partialID : Identity := ⟨4, 3⟩
def completeID : Identity := ⟨4, 4⟩

def nestedGraph (second : Identity) : Declarations := precedence ++ [
  ⟨nestedID, 1, .union nestedID [⟨branchA, "left", oid⟩,
    ⟨branchB, "right", oid⟩], none⟩,
  ⟨partialID, 0, .object [⟨fieldB, "other", "other", true, sid⟩] true, none⟩,
  ⟨completeID, 0, .object [] true, none⟩,
  ⟨outerID, 2, .union outerID [⟨branchA, "nested", nestedID⟩,
    ⟨branchB, "other", second⟩], none⟩]

/-- Nested ambiguity excludes that complete candidate. A separate complete
interpretation still wins before fallback ambiguity can be consulted. -/
theorem completeBranchSurvivesNestedAmbiguity :
    resolve (nestedGraph completeID) checks keys .authoredExample outerID
      (.object [("wire", .scalar (.string "ok"))]) =
      .ok ⟨.union outerID branchB
        (.object [] [("wire", .scalar (.string "ok"))]), []⟩ := by cbv

/-- With no complete branch, an ambiguity obstruction remains viable and
preferred. A nonpreferred incomplete branch cannot erase that obstruction. -/
theorem fallbackRetainsNestedAmbiguity :
    resolve (nestedGraph partialID) checks keys .authoredExample outerID
      (.object [("wire", .scalar (.string "ok"))]) = .error .ambiguous := by cbv

theorem partialNestedAmbiguityRetained :
    resolve (nestedGraph partialID) checks keys .authoredExample outerID
      (.object []) = .error .ambiguous := by cbv

/-- These controls were first observed failing: nested `return` in the body
bypassed the trailing enum gate. Body evaluation must finish before that gate. -/
theorem anyWholeEnumRejected :
    resolve [⟨aid, 0, .any, some [.any (.scalar (.string "allowed"))]⟩]
      checks keys .authoredExample aid (.scalar (.string "wrong")) = .error .invalid := by cbv

theorem arrayWholeEnumRejected :
    resolve [⟨oid, 0, .array oid {}, some [.array []]⟩]
      checks keys .authoredExample oid (.array [.array []]) = .error .invalid := by cbv

theorem objectWholeEnumRejected :
    resolve (scalars ++ [⟨oid, 0,
      .object [⟨fieldA, "source", "wire", true, sid⟩] false,
      some [.object [(fieldA, .scalar (.string "allowed"))] []]⟩])
      checks keys .authoredExample oid (.object [("wire", .scalar (.string "wrong"))]) =
      .error .invalid := by cbv

theorem mapWholeEnumRejected :
    resolve (scalars ++ [⟨oid, 0, .map (.scalar .string) {} sid {}, some [.map []]⟩])
      checks keys .authoredExample oid (.map [(.string "key", .scalar (.string "value"))]) =
      .error .invalid := by cbv

/-- The union gate already applied before the body/gate correction. Keep this
control alongside affected container cases so the repair cannot regress it. -/
theorem unionWholeEnumRejected :
    resolve (scalars ++ [⟨uid, 1, .union uid [⟨branchA, "text", sid⟩, ⟨branchB, "int", iid⟩],
      some [.union uid branchA (.scalar (.string "allowed"))]⟩])
      checks keys .authoredExample uid (.selected uid branchB (.scalar (.integer 1))) =
      .error .invalid := by cbv

/-- Class IDs are exact host DeepEqual evidence, supplied by the separately
checked adapter. They include concrete map and dynamic descendant types. -/
def integerHostMap (identity : Nat) : Input :=
  .host identity (.map [(.integer 1, .host 11 (.scalar (.string "one")))])
def integerHostValue (identity : Nat) : Value :=
  .host identity (.map [(.integer 1, .host 11 (.scalar (.string "one")))])
def hostMapEnum : Declarations :=
  [⟨aid, 0, .any, some [.any (integerHostValue 101)]⟩]

theorem hostMapEnumSelfAccepted :
    resolve hostMapEnum checks keys .enumMember aid (integerHostMap 101) =
      .ok ⟨.any (integerHostValue 101), []⟩ := by cbv

/-- int-key and int64-key maps have different host classes despite identical
normalized integer keys. The legacy non-string-map fallback is not numerical. -/
theorem hostMapConcreteTypeRejected :
    resolve hostMapEnum checks keys .authoredExample aid (integerHostMap 102) =
      .error .invalid := by cbv

/-- Dynamic int and int64 descendants differ in DeepEqual even in the same
map[int]any type. Host identity therefore covers the whole immutable payload. -/
theorem hostMapDynamicTypeRejected :
    rawAnyEqualAt 20
      (.host 201 (.map [(.integer 1, .host 21 (.scalar (.integer 1)))]))
      (.host 202 (.map [(.integer 1, .host 22 (.scalar (.integer 1)))])) = false := by cbv

theorem hostMapInsertionOrderAccepted :
    rawAnyEqualAt 20
      (.host 301 (.map [(.integer 1, .scalar (.string "one")),
        (.integer 2, .scalar (.string "two"))]))
      (.host 301 (.map [(.integer 2, .scalar (.string "two")),
        (.integer 1, .scalar (.string "one"))])) = true := by cbv

theorem hostBooleanMapSelfAccepted :
    resolve [⟨aid, 0, .any, some [.any (.host 401
      (.map [(.boolean true, .scalar (.string "yes"))]))]⟩]
      checks keys .enumMember aid (.host 401
        (.map [(.boolean true, .scalar (.string "yes"))])) =
      .ok ⟨.any (.host 401 (.map [(.boolean true, .scalar (.string "yes"))])), []⟩ := by cbv

/-- Matching class evidence never replaces raw payload validation. -/
theorem hostCollisionRejectedBeforeEnum :
    resolve [⟨aid, 0, .any, some [.any (.host 501 (.map []))]⟩]
      checks keys .authoredExample aid
      (.host 501 (.map [(.integer 1, .null), (.string "1", .null)])) =
      .error .invalid := by cbv

theorem hostDuplicateRejected :
    resolve scalars checks keys .authoredExample aid
      (.host 502 (.object [("x", .null), ("x", .null)])) = .error .invalid := by cbv

theorem hostCycleRejected :
    resolve scalars checks keys .authoredExample aid (.host 503 (.cycle 42)) =
      .error .cyclic := by cbv

theorem hostSelectedEvidenceRejectedByAny :
    resolve scalars checks keys .authoredExample aid
      (.host 504 (.selected uid branchA .null)) = .error .unsupported := by cbv

theorem hostDeclaredBytesCoerce :
    resolve scalars checks keys .authoredExample bid (.host 505 (.scalar (.string "hi"))) =
      .ok ⟨.scalar (.bytes [104, 105]), []⟩ := by cbv

theorem hostAliasPreservesAny :
    resolve (scalars ++ [⟨oid, 1, .alias aid, none⟩]) checks keys .authoredExample oid
      (integerHostMap 101) = .ok ⟨.any (integerHostValue 101), []⟩ := by cbv

theorem hostUnionBranchEvidencePreserved :
    resolve unionGraph checks keys .authoredExample uid
      (.host 506 (.selected uid branchB (integerHostMap 101))) =
      .ok ⟨.union uid branchB (.any (integerHostValue 101)), []⟩ := by cbv

theorem hostWrongOccurrenceRejected :
    resolve unionGraph checks keys .authoredExample uid
      (.host 507 (.selected oid branchB .null)) = .error .invalid := by cbv

def nonNullAnyGraph : Declarations := scalars ++ [
  ⟨oid, 1, .nonNull aid, none⟩, ⟨uid, 0, .array oid {}, none⟩]

theorem hostNonNullRejectsNull :
    resolve nonNullAnyGraph checks keys .authoredExample oid (.host 508 (.host 509 .null)) =
      .error .invalid := by cbv

theorem hostRequiredAnyElementRejectsNull :
    resolve nonNullAnyGraph checks keys .authoredExample uid
      (.host 510 (.array [.host 508 .null])) = .error .invalid := by cbv

theorem hostNullableRetainsNull :
    resolve (scalars ++ [⟨oid, 1, .nullable aid, none⟩]) checks keys .authoredExample oid
      (.host 508 .null) = .ok ⟨.null, []⟩ := by cbv

end ValueContract.Candidate.ResolutionControls
