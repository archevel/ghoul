package testpkg

// TestConflict is an exported constant in a test file that should NOT
// be picked up by the embalmer. If it includes it, the generated code
// will have a symbol that doesn't exist in the real package.
const TestConflict = 999
