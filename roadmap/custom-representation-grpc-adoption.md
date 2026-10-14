# Custom representation: gRPC consumer adoption (#926)

Steps 1–4 are complete with generated compilation, runtime races and exact
parent/candidate/repeat comparisons. Step 5 awaits the combined final review.

1. Trace shared default and field-initialization plans through protobuf
   constructors and metadata decoding, retaining their existing error channels.
2. Reproduce selected custom-default fallibility at the analysis seam before
   changing production. Distinguish existing unsupported opaque scalar casts
   from regressions introduced by the shared plan API migration.
3. Derive runtime failure ownership from actual selected conversion/default
   operations. Do not add UUID branches, swallow errors, or duplicate custom
   representation policy in gRPC. Coordinate any shared capability lowering
   with the generic conversion owner before extending transport support.
4. Verify supported custom/native controls with generated compilation and actual
   error-returning constructors/decoders. Preserve native output byte for byte;
   compare a bounded representative parent/candidate selection and independent
   candidate repeat. No exhaustive corpus.
5. Include this slice in the single independently reviewed #926 final diff.
