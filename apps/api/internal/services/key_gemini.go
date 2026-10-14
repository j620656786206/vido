package services

// KeyGemini is the optional Gemini key: subtitle translation on Gemini models
// (gemini-translation-dispatch) and filename parsing. Declared in its own file
// so the key kind lands beside the holders that use it; the resolver's
// switches know it.
const KeyGemini KeyName = "gemini"

// SecretNameGemini is the encrypted-store name of the Gemini key.
const SecretNameGemini = "gemini.api_key"
