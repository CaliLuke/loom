// Readiness-only actual JavaScript JSON decoding and raw-source revival.
// Pass the task-owned output directory from numeric_factor_probe.go.txt.
const fs = require("node:fs");
const path = require("node:path");

console.log(process.version);
for (const name of ["integer", "hex_quarter", "decimal_tiny", "hex_tiny", "float64_point_one", "float32_point_one"]) {
  const token = fs.readFileSync(path.join(process.argv[2], name + ".json"), "utf8");
  const native = JSON.parse(token);
  const raw = JSON.parse(token, (key, value, context) => context.source);
  if (raw !== token) {
    throw new Error("raw-source reviver changed " + name);
  }
  console.log(name, "native=" + native, "source-preserved=" + (raw === token));
}
