// Validate complete emitted schemas with JSON Schema 2020-12 and native ECMA-262
// regular expressions. No coercion, defaults, mutation or format inference.
const fs = require('node:fs');
const Ajv2020 = require('ajv/dist/2020').default;
const input = JSON.parse(fs.readFileSync(0, 'utf8'));
const ajv = new Ajv2020({strict: false, allErrors: true, validateFormats: false});
const output = input.map(({schema, instances}) => {
  const validate = ajv.compile(schema);
  return instances.map(value => {
    const valid = validate(value);
    return {valid, errors: valid ? [] : validate.errors};
  });
});
process.stdout.write(JSON.stringify(output));
