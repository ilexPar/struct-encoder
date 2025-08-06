```go
// TODO: clean this
generated, _ := json.Marshal(localRepresentations)
fmt.Printf("start: %s \n end \n", generated)

foreigns, _ := json.Marshal(foreignRepresentations)
fmt.Printf("start: %s \n end \n", foreigns)
```

```mermaid
classDiagram
  class TargetField {
    - []string Path
    - Kind Kind
  }
  TargetField  --*  SourceField
  class FieldTag {
    + []string RawOpts
    + string RawPath
    + []string Path
    + TagOpts Opts
  }
  TypeMatch  --*  FieldTag
  TagOpts  --*  FieldTag
  FieldTag  --*  SourceField
  class TagOpts {
    + []TypeMatch MatchTypes
  }
  class StructRef {
    + []SourceField Fields
    - string SourceType
    - string TargetType
  }
  SourceField  --*  StructRef
  StructRef  --*  SourceField
  class SourceField {
    - bool Skip
    - string Name
    - TargetField Target
    - StructureRef Child
    - bool IsPointer
    - bool IsArray
    - Kind Kind
  }
  class TypeMatch {
    + []string Path
    + string Name
    + bool Matches
  }

```
