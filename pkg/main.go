// Utility to "encode" a go struct into another using golang reflect.
//
// Intended to translate between API objects and internal system abstractions,
// providing capabilities to configure how each field should be mapped.
//
// # Usage
//
// The most basic usage is to annotate your struct fields with the `se` tag,
// specifying the "jsonpath" to the field in the destination object.
//
//	type MyStruct struct {
//	    Name string `se:metadata.name`
//	}
//
// And now you just need to call `Marshal` or `Unmarshal` to translate between your structs.
//
//	// load your struct with values from an external object
//	dst := &MyStruct{}
//	src := getSomeThirdPartyData()
//	se.Unarshal(src, dst)
//
//	// load an third party struct from you internal abstraction
//	dst := &module.SomeStruct{}
//	src := &MyStruct{
//	    Name: "pepito"
//	}
//	se.Marshal(src, dst)
//
// # Advanced
//
// # Type Matching
//
// By default types wont be checked, but you can specify type matching option like `types<SomeType>`.
// Multiple types can be annotated for each field using `|` as separator.
// When types don't match field will be skipped
//
// Example:
//
//	type MyStruct struct {
//	    Name string `se:metadata.name,types<SomeStruct|OtherStruct>`
//	    Flag bool `se:metadata.name,types<SomeStruct>`
//	}
//
// # Per Type Path
//
// You can specify a different path for each type by appending the path to the type using `:` as separator in the
// `types<>` option.
//
// Keep in mind:
//
//   - Is mandatory the main path for the field is set to `+` character when using this feature
//   - the `types<>` option will perform the match only based on the type you expect to "encode/decode",
//     so there's no need to know the types of fields disregarding the depth
//
// Example:
//
//	type MyStruct struct {
//	    Name string `se:+,types<SomeStruct:meta.name|OtherStruct:info.name>`
//	}
//
// # Nesting
//
// By default fields that are structs will inherit the parent path, but you can dismiss this by using
// the `->` operator as field name in order for the path to be fully processed
//
// Example:
//
//	type PreservedParent struct {
//	    Name `se:name`
//	}
//	type DismissParent struct {
//	    SomeProperty string `se:some.path.to.property`
//	}
//
//	type MyStruct struct {
//	    Child1 PreservedParent `se:some.path.to.use`
//	    Child2 DismissParent `->`
//	}
//
// # Introspection Caching
//
// Analyzed structs get cached to prevent unnecessary processing.
//
// You can preload introspection cache by calling `Introspect(local, foreign)`.
// Where `local` is the struct annotated with `se` tag, and `foreign` is the struct target for those tags
//
// In case of need cache can be cleared by calling `ClearTypeCache()`.
package pkg

const (
	// Operators
	//
	// field tag to parse
	FIELD_TAG_KEY = "se"
	// type separator when encoding to multiple types from a single source, eg se:"example,types<Struct1|Struct2>"
	TYPES_SPLIT = "|"
	// type path separator when setting per type path, eg se:"+,types<Struct1:path.one|Struct2:path.name>"
	TYPES_PATH_SPLIT = ":"
	// path to be used when dismissing path nesting
	DISMISS_NESTED = "->"
	// path name to be used when setting per type path, eg se:"+,types<Struct1:path.one|Struct2:path.name>"
	MULTI_TYPE_NAME = "+"

	TYPE_OPTS_REGEX = `^types<([^>]+)>$`
)

const (
	ErrUnmarshalDestType        = "into parameter must be a pointer to a non-nil struct"
	ErrUnmarshalSrcType         = "from parameter must be a struct or a pointer to a non nil struct"
	ErrLocalTypeNotStruct       = "local type must be a struct or non-nil pointer to a struct"
	ErrForeignTypeNotStruct     = "foreign type must be a struct or non-nil pointer to a struct"
	ErrLocalTypeMissingValidTag = "could not find any serializable field"
	ErrForeignTypeMissingField  = "field not found in path:"
	ErrForeignTypeMismatch      = "field type mismatch:"
	ErrInvalidPerTypePath       = "main path should be '+' when using per-type path matching"
)

// Unmarshal decodes a source object into a destination object using the struct mapping (sm) tags.
// The `from` parameter is the source object to decode from, which must be a struct or a pointer to a non-nil struct.
// The `into` parameter is the destination object to decode into, which must be a pointer to a non-nil struct.
// Fields are mapped according to the sm tag rules defined in the package documentation.
// Returns an error if the types are invalid or if the decoding process fails.
func Unmarshal(from interface{}, into interface{}) error {
	cacheInit()
	decoder := &StructDecoder{}
	if err := decoder.init(from, into); err != nil {
		return err
	}

	return decoder.run()
}

// Marshal encodes a source object into a destination object using the struct mapping (sm) tags.
// The `from` parameter is the source object to encode from, which must be a struct or a pointer to a non-nil struct.
// The `into` parameter is the destination object to encode into, which must be a pointer to a non-nil struct.
// Fields are mapped according to the sm tag rules defined in the package documentation.
// Returns an error if the types are invalid or if the encoding process fails.
func Marshal(from interface{}, into interface{}) error {
	cacheInit()
	encoder := &StructEncoder{}
	if err := encoder.init(from, into); err != nil {
		return err
	}
	return encoder.run()
}
