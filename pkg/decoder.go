package pkg

import (
	"errors"
	"reflect"
)

// StructDecoder provides functionality for decoding data between struct types
// using reflection. It maps fields from a "foreign" struct to a "local" struct
// based on tag information.
//
// The decoder handles nested structs, pointers, slices, and various type
// conversions while maintaining proper error handling for nil values and
// type incompatibilities.
type StructDecoder struct {
	local          reflect.Value // destination for a decoder
	foreign        reflect.Value // source for a decoder
	representation *StructRepr
}

func (this *StructDecoder) validateInput() error {
	if this.local.Kind() != reflect.Pointer || this.local.IsNil() {
		return errors.New(ErrUnmarshalDestType)
	}

	if this.foreign.Kind() == reflect.Ptr && this.foreign.IsNil() {
		return errors.New(ErrUnmarshalSrcType)
	}

	return nil
}

// StructDecoder decodes data from a "foreign" struct into a "local" struct.
// It uses reflection to map fields between the two structs based on tag information
// maintained in a StructRepr representation.
//
// The decoder follows these steps:
// 1. Validates that inputs are appropriate struct types
// 2. Generates a field mapping representation
// 3. Copies values from the foreign struct to the local struct
//
// It handles nested structs, pointers, and slices with special consideration
// for nil values and type compatibility.
func (this *StructDecoder) init(foreign interface{}, local interface{}) error {
	this.local = reflect.ValueOf(local)
	this.foreign = reflect.ValueOf(foreign)

	if err := this.validateInput(); err != nil {
		return err
	}

	this.representation = &StructRepr{}
	err := this.representation.introspect(local, foreign)
	if err != nil {
		return this.unwrapIntrospectErr(err)
	}

	return nil
}

func (this StructDecoder) unwrapIntrospectErr(err error) error {
	msg := err.Error()

	if msg == ErrLocalTypeNotStruct {
		return errors.New(ErrUnmarshalDestType)
	}

	if msg == ErrForeignTypeNotStruct {
		return errors.New(ErrUnmarshalSrcType)
	}

	return err
}

func (this *StructDecoder) run() error {
	return setLocalFieldsValue(this.foreign, this.local, this.representation.Fields)
}

// setLocalFieldsValue recursively copies data from source fields to destination fields based on mapping information.
//
// Parameters:
//   - src: The source (foreign) value to copy data from
//   - dst: The destination (local) value to copy data to
//   - reprFields: The field mapping rules that define how to copy between structures
//
// The function handles:
//   - Dereferencing pointers in both source and destination
//   - Creating new objects for nil destination pointers
//   - Handling nested struct fields by recursion
//   - Array/slice fields in structures
//   - Reading data from source fields using provided field indices
//
// Returns an error if accessing or setting field values fails.
func setLocalFieldsValue(src, dst reflect.Value, reprFields []SourceField) error {
	source := src
	if source.Kind() == reflect.Ptr {
		source = source.Elem()
	}

	target := dst
	targetKind := target.Kind()
	if targetKind == reflect.Ptr {
		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}
		target = target.Elem()
	}

	for _, field := range reprFields {
		child, hasChild := localRepresentations[field.ChildRef]
		foreign := foreignRepresentations[field.TargetRef]

		if hasChild {
			err := setLocalChildValue(field, child, src, target)
			if err != nil {
				return err
			}
		} else {
			data, err := getForeignFieldData(foreign.IndexPath, source)
			if err != nil {
				return err
			}
			if data != nil {
				target.Field(field.Id).Set(reflect.ValueOf(data))
			}
		}
	}

	return nil
}

// setLocalChildValue handles setting values for nested struct fields (child structures)
// within the local struct. It creates or updates nested struct fields based on field mapping.
//
// Parameters:
//   - field: The field mapping information containing type and relationship details
//   - child: The representation of the child struct with its own field mappings
//   - src: The source (foreign) value that may contain data for the child structure
//   - target: The target (local) value where the child structure should be populated
//
// The function supports:
//   - Creating and populating slices of structs when field.IsArray is true
//   - Setting values on direct struct fields when field.IsArray is false
//   - Recursively populating child structs by calling setLocalFieldsValue
//
// Returns an error if setting field values fails during recursion.
func setLocalChildValue(field SourceField, child StructRepr, src, target reflect.Value) error {
	var childTarget reflect.Value
	if field.IsArray && field.Kind == reflect.Struct {
		slice := reflect.MakeSlice(field.Type, 0, 1)
		ptr := reflect.New(field.Type.Elem())
		slice = reflect.Append(slice, ptr.Elem())
		target.Field(field.Id).Set(slice)
		childTarget = target.Field(field.Id).Index(0)
	}

	if field.Kind == reflect.Struct {
		if !field.IsArray {
			childTarget = target.Field(field.Id)
		}
		err := setLocalFieldsValue(src, childTarget, child.Fields)
		if err != nil {
			return err
		}
	}
	return nil
}

// descendIntoForeignArrayField traverses into array or slice fields in foreign structures.
//
// Parameters:
//   - from: The source value to examine, which may be an array or slice
//   - finalValue: Whether this is the final value being accessed in a field path
//
// Returns:
//   - A reflect.Value that is either the same as the input (if not an array/slice),
//     or the first element of the array/slice
//   - A boolean indicating whether processing should be skipped (true for nil or empty arrays)
//
// This function handles special cases for arrays and slices:
// - Returns a skip flag for nil or empty collections
// - For non-empty collections, returns the first element
// - Has special behavior when the collection contains struct elements and is the final value
func descendIntoForeignArrayField(from reflect.Value, finalValue bool) (reflect.Value, bool) {
	kind := from.Kind()
	if kind == reflect.Slice || kind == reflect.Array {
		if from.IsNil() {
			return from, true
		}

		if from.Len() > 0 {
			// TODO: support non 0 indexes maybe?
			if finalValue {
				sliceElemKind := from.Type().Elem().Kind()
				if sliceElemKind == reflect.Struct {
					return from.Index(0), false
				}
			} else {
				return from.Index(0), false
			}
		} else {
			return from, true
		}
	}

	return from, false
}

// getForeignFieldData extracts data from a nested field path in a foreign structure.
//
// Parameters:
//   - fieldIndexes: An array of field indices representing the path to the desired field
//   - from: The source value to extract data from
//
// Returns:
//   - interface{}: The extracted field value, or nil if the field is nil/zero/invalid
//   - error: Any error encountered during the extraction process
//
// This function navigates through a structure following the provided field indices path.
// It handles:
//   - Pointer dereferencing
//   - Array/slice traversal (using descendIntoForeignArrayField)
//   - Nested field access
//   - Nil pointer and zero value detection
//
// When it reaches the final field in the path, it returns the field's interface value.
// If any field along the path is nil, invalid, or zero, nil is returned.
func getForeignFieldData(fieldIndexes []int, from reflect.Value) (interface{}, error) {
	if from.Kind() == reflect.Pointer {
		from = from.Elem()
	}

	// evaluation order is highly important
	for idx, fieldId := range fieldIndexes {
		var skip bool
		from, skip = descendIntoForeignArrayField(from, idx == len(fieldIndexes)-1)
		if skip {
			return nil, nil
		}

		if from.Kind() == reflect.Pointer {
			if from.IsNil() {
				// Ignore nil pointers
				return nil, nil
			}
			from = from.Elem()
		}

		from = from.Field(fieldId)

		if idx == len(fieldIndexes)-1 {
			if !from.IsValid() || from.IsZero() {
				return nil, nil
			}
			return from.Interface(), nil
		}
	}

	return nil, nil
}
