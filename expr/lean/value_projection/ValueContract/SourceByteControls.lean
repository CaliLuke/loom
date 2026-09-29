import ValueContract.SourceAdmissionControls
import ValueContract.Materialization

namespace ValueContract.Candidate.SourceByteControls
open ResolutionControls

/-- One declaration graph admits bytes, typed arrays, raw Any and their union.
Every branch sees the same original sequence evidence. -/
def declarations : Declarations := [
  ⟨iid, 0, .scalar .integer {sourcePrimitive := some .uint}, none⟩,
  ⟨bid, 0, .scalar .bytes {sourcePrimitive := some .bytes}, none⟩,
  ⟨aid, 0, .array iid {}, none⟩,
  ⟨sid, 0, .any, none⟩,
  ⟨oid, 0, .array sid {}, none⟩,
  ⟨uid, 1, .union uid [⟨branchA, "bytes", bid⟩, ⟨branchB, "array", aid⟩], none⟩]

def items : List (Nat × UInt8) := [(17, 104), (19, 105)]

theorem bytesAndArraysShareNativeSlice (role : Role) :
    resolve declarations checks numbers role bid (.byteSequence (.slice false items)) =
      .ok ⟨.scalar (.bytes [104, 105]), []⟩ ∧
    resolve declarations checks numbers role aid (.byteSequence (.slice false items)) =
      .ok ⟨.array [.scalar (.integer 104), .scalar (.integer 105)], []⟩ := by
  cases role <;> cbv

theorem nativeSliceUnionRemainsAmbiguous (role : Role) :
    resolve declarations checks numbers role uid (.byteSequence (.slice false items)) =
      .error .ambiguous := by
  cases role <;> cbv

/-- Erasing sequence shape into a byte scalar hides a valid Array candidate.
The normalized scalar input is retained only as the old lossy counterexample. -/
theorem scalarErasureLosesCompetingArray (role : Role) :
    resolve declarations checks numbers role uid (.scalar ⟨.bytes [104, 105], .nativeBytes⟩) =
      .ok ⟨.union uid branchA (.scalar (.bytes [104, 105])), []⟩ ∧
    resolve declarations checks numbers role uid (.byteSequence (.slice false items)) =
      .error .ambiguous := by
  cases role <;> cbv

theorem fixedAndNamedSequencesSelectArray (role : Role) :
    resolve declarations checks numbers role uid (.byteSequence (.array false items)) =
      .ok ⟨.union uid branchB (.array [.scalar (.integer 104), .scalar (.integer 105)]), []⟩ ∧
    resolve declarations checks numbers role uid (.byteSequence (.slice true items)) =
      .ok ⟨.union uid branchB (.array [.scalar (.integer 104), .scalar (.integer 105)]), []⟩ := by
  cases role <;> cbv
  all_goals exact ⟨rfl, rfl⟩

theorem arrayAnyRetainsOriginalChildEvidence (role : Role) :
    resolve declarations checks numbers role oid (.byteSequence (.array true items)) =
      .ok ⟨.array [.any (.host 17 (.scalar (.integer 104))),
        .any (.host 19 (.scalar (.integer 105)))], []⟩ := by
  cases role <;> cbv

theorem rawAnyRetainsBinaryMeaning (role : Role) :
    resolve declarations checks numbers role sid (.host 23 (.byteSequence (.array true items))) =
      .ok ⟨.any (.host 23 (.scalar (.bytes [104, 105]))), []⟩ := by
  cases role <;> cbv

theorem rawBinaryMaterialization (codecs : ScalarCodecs) :
    materialize codecs (.host 23 (.scalar (.bytes [104, 105]))) =
      .ok (.text (codecs.bytes.encode [104, 105])) := by
  simp [materialize, encodeScalar, jsonValid]

theorem nilAndEmptyPreservePresence (role : Role) :
    resolve declarations checks numbers role aid (.byteSequence (.nilSlice false)) =
      .ok ⟨.nilArray, []⟩ ∧
    resolve declarations checks numbers role aid (.byteSequence (.slice false [])) =
      .ok ⟨.array [], []⟩ ∧
    resolve declarations checks numbers role sid (.byteSequence (.nilSlice true)) =
      .ok ⟨.any .nilBytes, []⟩ := by
  cases role <;> cbv

/-- Defined byte elements stay ordinary recursive inputs. Empty arrays remain
vacuously compatible; nonempty numeric arrays reject defined scalar children. -/
theorem definedElementsAreNotNativeBytes (role : Role) :
    resolve declarations checks numbers role aid
      (.array [.host 31 (.scalar ⟨.integer 104 7, .definedScalar .integer⟩)]) =
      .error .invalid ∧
    resolve declarations checks numbers role aid (.array []) = .ok ⟨.array [], []⟩ := by
  cases role <;> cbv

end ValueContract.Candidate.SourceByteControls
