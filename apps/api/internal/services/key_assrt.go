package services

// KeyAssrt is the optional Assrt (射手網) token: online Chinese subtitle search
// (disc-2026-11-assrt-key-in-settings). Declared in its own file like
// KeyGemini; the resolver's switches know it.
const KeyAssrt KeyName = "assrt"

// SecretNameAssrt is the encrypted-store name of the Assrt token. It keeps the
// name the Assrt provider has read since Story 8.x instead of the
// `{provider}.{field}` convention, so a token stored before this story still
// resolves without a data migration.
const SecretNameAssrt = "assrt_api_key"
