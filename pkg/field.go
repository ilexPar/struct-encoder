package pkg

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
)

// StructRepr represents a structure representation that stores information about
// struct fields and their corresponding foreign type. It is used to facilitate
// mapping between different struct types in the application.
type StructRepr struct {
	Fields          []SourceField
	ForeignRootType string
}

// SourceField represents a field in the source structure that needs to be mapped to
// a target structure. It contains information about the field such as its ID, name,
// type details, and references to child structures or target mappings.
//
// The SourceField is used by the StructRepr to define how fields from one structure
// should be mapped to fields in another structure, tracking properties like whether
// the field is a pointer, array, map, or has nested structures.
type SourceField struct {
	Id        int
	Name      string
	Kind      reflect.Kind
	Type      reflect.Type
	IsPointer bool
	IsArray   bool
	IsMap     bool
	ChildRef  string
	TargetRef string
	Tag       FieldTag
}

// TargetField represents a field in the target structure that will receive mapped data.
// It contains information necessary to locate and manipulate the target field,
// such as its ID, kind (type), path in the struct hierarchy, and index path.
//
// TargetField is used to store information about how to access specific fields
// in the target structure during the mapping process, tracking both the string path
// representation and the numerical indices needed for reflection-based access.
type TargetField struct {
	Id        int
	Kind      reflect.Kind
	Path      []string
	IndexPath []int
	TypeName  string
}

// describe creates a new StructRepr instance by analyzing the provided local and foreign types.
// It maps the fields of the local type to corresponding fields in the foreign type based on
// struct tags and field names.
//
// Parameters:
//   - local: The reflect.Type of the source structure to be analyzed.
//   - foreign: The reflect.Type of the target structure that fields will be mapped to.
//   - name: A name identifier for the representation, typically the field name in a parent struct.
//   - parentPath: Optional path elements that indicate the hierarchical location in nested structures.
//
// Returns:
//   - error: An error if the mapping generation fails, nil on success.
//
// The function uses caching to avoid redundant analysis of previously processed type combinations.
// It handles pointer types, resolves nested structures, and validates type compatibility between
// mapped fields.
func (this *StructRepr) describe(
	local, foreign reflect.Type,
	name string,
	parentPath ...string,
) error {
	if foreign.Kind() == reflect.Pointer {
		foreign = foreign.Elem()
	}
	this.ForeignRootType = foreign.Name()

	if local.Kind() == reflect.Pointer {
		local = local.Elem()
	}

	key := getNativeRepresentationKey(local, foreign, name)
	cached, ok := localRepresentations[key]
	if ok {
		*this = cached
		return nil
	}

	fields, err := parseStructFields(local, foreign, this.ForeignRootType, parentPath...)
	if err != nil {
		return err
	}

	this.Fields = fields
	localRepresentations[key] = *this

	return nil
}

// introspect analyzes the provided local and foreign interface{} objects and creates a mapping
// between their struct fields based on tags and field names.
//
// Parameters:
//   - local: The source object whose structure will be analyzed for mapping.
//   - foreign: The target object whose structure will receive mapped data.
//
// Returns:
//   - error: An error if the introspection process fails, nil on success.
//
// The function validates that both local and foreign are struct types, initializes the caching
// system, and then describes the relationship between the structures. It also ensures that
// at least one valid mapping field exists between the structures.
func (this *StructRepr) introspect(local, foreign interface{}) error {
	l := reflect.TypeOf(local)
	f := reflect.TypeOf(foreign)

	if l.Kind() == reflect.Pointer {
		l = l.Elem()
	}

	if f.Kind() == reflect.Pointer {
		f = f.Elem()
	}

	cacheInit()

	if err := this.validateInput(l, f); err != nil {
		return err
	}

	if err := this.describe(l, f, ""); err != nil {
		return err
	}

	if len(this.Fields) == 0 {
		return errors.New(ErrLocalTypeMissingValidTag)
	}

	return nil
}

func (this StructRepr) validateInput(local, foreign reflect.Type) error {
	if local.Kind() != reflect.Struct {
		return errors.New(ErrLocalTypeNotStruct)
	}
	if foreign.Kind() != reflect.Struct {
		return errors.New(ErrForeignTypeNotStruct)
	}
	return nil
}

// findFieldChilds analyzes a source field to determine if it contains a nested structure that
// needs to be represented separately. It creates and caches representations for nested structs.
//
// Parameters:
//   - field: The SourceField to analyze for nested structures.
//   - stfield: The reflect.StructField from the original structure type definition.
//   - foreign: The target foreign type that fields will be mapped to.
//   - parentPath: The path elements indicating the hierarchical location of this field.
//
// Returns:
//   - string: A key identifying the nested structure representation in the cache, or empty if none.
//   - error: An error if generation of the nested representation fails, nil otherwise.
//
// The function determines if a field is a struct, or contains a struct (through a pointer, array, or map),
// and generates a representation for that nested structure. It uses caching to avoid redundant
// analysis of previously processed struct types.
func findFieldChilds(
	field SourceField,
	stfield reflect.StructField,
	foreign reflect.Type,
	parentPath []string,
) (string, error) {
	var pregnant bool // identify if the field is a struct to create its representation
	var childRef reflect.Type
	var err error
	key := ""

	if field.Kind == reflect.Struct {
		pregnant = true
		childRef = stfield.Type
	}

	if field.IsPointer || field.IsArray || field.IsMap {
		if stfield.Type.Elem().Kind() == reflect.Struct {
			childRef = stfield.Type.Elem()
		}
	}

	if pregnant {
		key = getNativeRepresentationKey(childRef, foreign, field.Name)
		_, ok := localRepresentations[key]
		if ok {
			return key, nil
		}
		repr := &StructRepr{}
		err = repr.describe(childRef, foreign, field.Name, parentPath...)
		localRepresentations[key] = *repr
	}

	return key, err
}

func newField(id int, stfield reflect.StructField, tag FieldTag, target string) SourceField {
	kind := stfield.Type.Kind()
	field := SourceField{
		Id:        id,
		Name:      stfield.Name,
		Kind:      kind,
		Type:      stfield.Type,
		IsPointer: kind == reflect.Pointer,
		IsArray:   kind == reflect.Array || kind == reflect.Slice,
		IsMap:     kind == reflect.Map,
		Tag:       tag,
		TargetRef: target,
	}

	if field.IsArray || field.IsMap || field.IsPointer {
		field.Kind = stfield.Type.Elem().Kind()
	}

	return field
}

// parseStructFields analyzes a local struct type and creates a mapping to fields in a foreign struct type.
// It iterates through each field in the local struct, extracting tag information, finding matching target
// fields in the foreign struct, and validating type compatibility.
//
// Parameters:
//   - local: The reflect.Type of the source structure to be analyzed.
//   - foreign: The reflect.Type of the target structure that fields will be mapped to.
//   - foreignRootType: The name of the root type of the foreign structure.
//   - parentPath: Optional path elements that indicate the hierarchical location in nested structures.
//
// Returns:
//   - []SourceField: A slice of SourceField structs representing the mappable fields from the local struct.
//   - error: An error if the mapping generation fails, nil on success.
//
// The function processes each field to determine its mapping characteristics, skips fields marked
// with the skip tag, validates type compatibility for direct field mappings, and identifies nested
// structures that require their own mapping representations.
func parseStructFields(
	local, foreign reflect.Type,
	foreignRootType string,
	parentPath ...string,
) ([]SourceField, error) {
	fields := make([]SourceField, 0)
	for id := range local.NumField() {
		stfield := local.Field(id)
		tag, target, err := getTagAndTarget(foreignRootType, stfield, foreign, parentPath)
		if tag.Skip {
			continue
		}
		if err != nil {
			return nil, err
		}

		field := newField(id, stfield, tag, target)
		childRef, err := findFieldChilds(field, stfield, foreign, tag.Path)
		if err != nil {
			return nil, err
		}

		field.ChildRef = childRef
		if field.ChildRef == "" {
			// having no children means we will write over this field
			// make sure Local and Foreign fields types matches
			err := validateFieldsTypeMatch(field, stfield, tag.TargetType)
			if err != nil {
				return nil, err
			}
		}

		fields = append(fields, field)
	}

	return fields, nil
}

// validateFieldsTypeMatch checks if the type of a source field matches the expected target type.
// This validation ensures type compatibility during mapping operations between structures.
//
// Parameters:
//   - field: The SourceField being validated.
//   - stfield: The reflect.StructField from the original structure type definition.
//   - targetType: The expected type name in the target structure.
//
// Returns:
//   - error: An error if the types don't match, nil otherwise.
//
// The function handles different field kinds (regular, array, map, pointer) and verifies that
// the underlying type matches the expected target type. If the field points to or is a struct,
// the validation is skipped as struct mappings are handled separately.
func validateFieldsTypeMatch(
	field SourceField,
	stfield reflect.StructField,
	targetType string,
) error {
	typeName := stfield.Type.Name()
	if field.IsArray || field.IsMap || field.IsPointer {
		typeName = stfield.Type.Elem().Name()
	}
	if !pointsOrIsStruct(stfield.Type) && targetType != typeName {
		return fmt.Errorf(ErrForeignTypeMismatch+" %v is not %v", targetType, typeName)
	}
	return nil
}

// pointsOrIsStruct determines whether a given reflect.Type is a struct type or points to a struct type.
// This function is used to identify fields that contain or reference structured data.
//
// Parameters:
//   - field: The reflect.Type to check.
//
// Returns:
//   - bool: true if the field is a struct or points to a struct (via pointer, array, or slice),
//     false otherwise.
//
// The function first checks if the field is directly a struct type. If not, it checks if the field
// is of a kind that can contain other types (pointer, array, slice) and if the element type is a struct.
func pointsOrIsStruct(field reflect.Type) bool {
	descendableKinds := []reflect.Kind{reflect.Pointer, reflect.Array, reflect.Slice}
	kind := field.Kind()
	if kind == reflect.Struct {
		return true
	}
	if slices.Contains(descendableKinds, kind) && field.Elem().Kind() == reflect.Struct {
		return true
	}
	return false
}

// Introspect analyzes the provided local and foreign interface{} objects and creates a structural
// representation of how fields from the local structure can be mapped to the foreign structure.
//
// Parameters:
//   - local: The source object whose structure will be analyzed for mapping based on `se` tag.
//   - foreign: The target object whose structure will receive mapped data.
//
// Returns:
//   - error: An error if the introspection process fails, nil on success.
//
// Introspect validates that both local and foreign are struct types, analyzes their fields,
// and generates a mapping representation that can be used for data transfer between them.
// It uses reflection to examine structure types, field tags, and type compatibility,
// creating a cached representation to optimize repeated mappings.
func Introspect(local, foreign interface{}) error {
	repr := &StructRepr{}
	return repr.introspect(local, foreign)
}
