package main

// EmbalmEntry represents a package to be mummified, parsed from graveyard.toml.
type EmbalmEntry struct {
	Package         string `toml:"package"`
	SkipUnwrappable bool   `toml:"skip_unwrappable"`
}

// Graveyard is the top-level structure of graveyard.toml.
type Graveyard struct {
	Embalm []EmbalmEntry `toml:"embalm"`
}

// defaultStdlib is the set of stdlib packages included by default,
// matching the current sarcophagus.txt. All default to skip_unwrappable = true
// since stdlib packages may contain channels/maps.
var defaultStdlib = []EmbalmEntry{
	{Package: "math", SkipUnwrappable: true},
	{Package: "strings", SkipUnwrappable: true},
	{Package: "strconv", SkipUnwrappable: true},
	{Package: "os", SkipUnwrappable: true},
	{Package: "path/filepath", SkipUnwrappable: true},
	{Package: "fmt", SkipUnwrappable: true},
	{Package: "io", SkipUnwrappable: true},
	{Package: "time", SkipUnwrappable: true},
	{Package: "regexp", SkipUnwrappable: true},
	{Package: "sort", SkipUnwrappable: true},
	{Package: "encoding/json", SkipUnwrappable: true},
	{Package: "net/http", SkipUnwrappable: true},
	{Package: "crypto/sha256", SkipUnwrappable: true},
	{Package: "crypto/md5", SkipUnwrappable: true},
	{Package: "encoding/base64", SkipUnwrappable: true},
	{Package: "encoding/hex", SkipUnwrappable: true},
	{Package: "net/url", SkipUnwrappable: true},
	{Package: "bytes", SkipUnwrappable: true},
	{Package: "bufio", SkipUnwrappable: true},
	{Package: "errors", SkipUnwrappable: true},
	{Package: "unicode", SkipUnwrappable: true},
	{Package: "unicode/utf8", SkipUnwrappable: true},
}
