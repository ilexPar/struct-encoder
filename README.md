# struct-encoder
Utility to "encode" a go struct into another using golang reflect.

Intended to translate between API objects and internal system abstractions, providing capabilities to configure how each field should be mapped.

## Usage

The most basic usage is to annotate your struct fields with the `se` tag, specifying the "jsonpath" to the field in the destination object.

```go
type MyStruct struct {
    Name string `se:metadata.name`
}
```

And now you just need to call `Marshal` or `Unmarshal` to translate between your structs.

```go
// load your struct with values from an external object
dst := &MyStruct{}
src := getSomeThirdPartyData()
se.Unarshal(src, dst)

// load an third party struct from you internal abstraction
dst := &module.SomeStruct{}
src := &MyStruct{
    Name: "pepito"
}
se.Marshal(src, dst)
```

## Advanced


### Type Matching
By default types wont be checked, but you can specify type matching option like `types<SomeType>`. Multiple types can be annotated for each field using `|` as separator.
When types don't match field will be skipped

Example:

```go
type MyStruct struct {
    Name string `se:metadata.name,types<TypeOne|TypeTwo>`
    Flag bool `se:metadata.name,types<TypeOne>`
}
```

### Per Type Path

You can specify a different path for each type by appending the path to the type using `:` as separator in the `types<>` option.

Keep in mind:
- Is mandatory the main path for the field is set to `+` character when using this feature
- the `types<>` option will perform the match only based on the type you expect to "encode/decode", so there's no need to know the types of fields disregarding the depth

Example:

```go
type MyStruct struct {
    Name string `se:+,types<SomeStruct:meta.name|OtherStruct:info.name>`
}
```

### Nesting

By default fields that are structs will inherit the parent path, but you can dismiss this by using the `->` operator as field name in order for the path to be fully processed

Example:

```go
type PreservedParent struct {
    Name `se:name`
}
type DismissParent struct {
    SomeProperty string `se:some.path.to.property`
}

type MyStruct struct {
    Child1 PreservedParent `se:some.path.to.use`
    Child2 DismissParent `->`
}

```

## Introspection Caching

Analysed structs get cached to prevent unnecessary processing.

You can preload introspection cache by calling `Introspect(local, foreign)`.
Where `local` is the struct annotated with `se` tag, and `foreign` is the struct target for those tags

In case of need you can clear the cache by calling `ClearTypeCache()`.
