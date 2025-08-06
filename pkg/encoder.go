package pkg

import (
	"errors"
	"reflect"
)

// StructEncoder is a utility for encoding data from a local (source) structure to a foreign (destination) structure.
// It uses reflection to map fields between the two structures based on predefined representation rules.
// The encoder validates that the source contains valid data and the destination is a valid pointer to a struct
// before performing the encoding operation.
type StructEncoder struct {
	local          reflect.Value
	foreign        reflect.Value
	representation *StructRepr
}

func (this *StructEncoder) validateInput() error {
	if this.foreign.Kind() != reflect.Pointer || this.foreign.IsNil() {
		return errors.New(ErrUnmarshalDestType)
	}

	if this.local.Kind() == reflect.Ptr && this.local.IsNil() {
		return errors.New(ErrUnmarshalSrcType)
	}

	return nil
}

// init initializes the StructEncoder with the provided local (source) and foreign (destination) interfaces.
// It performs validation on the input types and generates the representation rules for encoding.
//
// Parameters:
//   - local: The source struct or pointer to struct containing the data to be encoded
//   - foreign: A pointer to the destination struct where data will be encoded to
//
// Returns:
//   - error: Any validation or initialization error that occurred
//
// The encoder follows these steps:
// 1. Validates that inputs are appropriate struct types
// 2. Generates a field mapping representation
// 3. Copies values from the local struct to the foreign struct
func (this *StructEncoder) init(local interface{}, foreign interface{}) error {
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

func (this *StructEncoder) unwrapIntrospectErr(err error) error {
	msg := err.Error()

	if msg == ErrForeignTypeNotStruct {
		return errors.New(ErrUnmarshalDestType)
	}

	if msg == ErrLocalTypeNotStruct {
		return errors.New(ErrUnmarshalSrcType)
	}

	return err
}

func (this *StructEncoder) run() error {
	return setForeingFieldsValue(this.local, this.foreign, this.representation.Fields)
}

// setForeignFieldsValue copies values from the source struct to the destination struct
// based on the field mapping defined in the representation rules.
//
// Parameters:
//   - src: The reflect.Value of the source struct containing the data to be encoded
//   - dst: The reflect.Value of the destination struct where data will be encoded to
//   - reprFields: A slice of SourceField structs that define the mapping between fields
//
// Returns:
//   - error: Any error that occurred during the operation
//
// The function:
// 1. Handles pointer dereferencing for both source and destination
// 2. Properly processes slices and arrays (taking first element if non-empty)
// 3. Iterates through each field in the representation mapping
// 4. For nested structs, recursively calls itself
// 5. For simple fields, uses setForeignFieldData to copy the value
//
// This function is the core of the encoding process, mapping source values to
// their corresponding destination fields according to the predefined representation.
func setForeingFieldsValue(src, dst reflect.Value, reprFields []SourceField) error {
	source, empty := digIntoLocalSource(src)
	if empty {
		return nil
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
			err := setForeingFieldsValue(source.Field(field.Id), target, child.Fields)
			if err != nil {
				return err
			}
		} else {
			data := source.Field(field.Id)
			err := setForeignFieldData(foreign.IndexPath, target, data)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// digIntoLocalSource handles pointer and collection types in the source value.
// It dereferences pointers and extracts the first element from slices/arrays.
//
// Parameters:
//   - source: The reflect.Value to be processed
//
// Returns:
//   - reflect.Value: The processed value after dereferencing and element extraction
//   - bool: True if the source is empty (nil pointer or empty collection), false otherwise
//
// The function:
// 1. Dereferences pointers (returning empty flag if nil)
// 2. For slices and arrays, extracts the first element (returning empty flag if length is 0)
// 3. Returns the resulting value and whether it should be considered empty
//
// This is used to normalize source values before mapping them to destination fields,
// ensuring consistent handling of various input types.
func digIntoLocalSource(source reflect.Value) (reflect.Value, bool) {
	if source.Kind() == reflect.Ptr {
		if source.IsNil() {
			return source, true // ignore empty pointers
		}
		source = source.Elem()
	}
	if source.Kind() == reflect.Slice || source.Kind() == reflect.Array {
		if source.Len() == 0 {
			return source, true // ignore empty slices
		}
		source = source.Index(0)
	}
	return source, false
}

// setForeignFieldData sets a value from the source struct to a field in the destination struct.
// It handles various types including pointers, maps, arrays, and slices.
//
// Parameters:
//   - path: A slice of field indices representing the path to the target field in the destination struct
//   - target: The reflect.Value of the destination struct
//   - data: The reflect.Value containing the data to be set
//
// Returns:
//   - error: Any error that occurred during the operation
//
// The function follows these steps:
// 1. Checks if the data is valid (not zero or nil)
// 2. Handles pointer dereferencing for both source and destination
// 3. Navigates through the path to the target field
// 4. Creates necessary structures (slices, maps) if they don't exist
// 5. Sets the data to the target field
func setForeignFieldData(path []int, target reflect.Value, data reflect.Value) error {
	data, empty := digIntoLocalData(data)
	if empty {
		return nil
	}

	// handle target if pointer
	if target.Kind() == reflect.Ptr {
		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}
		target = target.Elem()
	}

	dst := target
	for idx, fieldId := range path {
		dst = descendIntoLocalArrayField(dst)

		if dst.Kind() == reflect.Pointer {
			if dst.IsNil() {
				dst.Set(reflect.New(dst.Type().Elem()))
			}
			dst = dst.Elem()
		}
		dst = dst.Field(fieldId)

		if idx == len(path)-1 {
			dst.Set(data)
			return nil
		}
	}

	return nil
}

// digIntoLocalData handles nil and zero values in the source data.
// It dereferences pointers and checks if the data is valid for processing.
//
// Parameters:
//   - data: The reflect.Value to be processed
//
// Returns:
//   - reflect.Value: The processed value after dereferencing
//   - bool: True if the data is empty (invalid, zero, or nil pointer), false otherwise
//
// The function:
// 1. Checks if the value is valid or zero (returning empty flag if invalid/zero)
// 2. Dereferences pointers (returning empty flag if nil)
// 3. Returns the resulting value and whether it should be considered empty
//
// This is used to ensure that only valid, non-empty values are set on the destination,
// avoiding attempts to set invalid or zero values which could cause errors.
func digIntoLocalData(data reflect.Value) (reflect.Value, bool) {
	if !data.IsValid() || data.IsZero() {
		return data, true
	}
	if data.Kind() == reflect.Ptr {
		if data.IsNil() {
			return data, true // empty data, do nothing
		}
		data = data.Elem()
	}
	return data, false
}

// descendIntoLocalArrayField handles complex data structures like maps, arrays, and slices
// when traversing through nested fields.
//
// Parameters:
//   - dst: The reflect.Value representing the current destination field
//
// Returns:
//   - reflect.Value: That is either same as the input (if not an array/slice)
//     or the first element of the array/slice
//
// The function:
// 1. Checks if the current value is a map, array, or slice
// 2. If empty/nil, initializes it appropriately
// 3. For empty collections, creates and appends a new element
// 4. Returns the first element for further traversal
//
// This allows the encoder to properly navigate through nested collections
// while ensuring all necessary structures are created along the path.
func descendIntoLocalArrayField(dst reflect.Value) reflect.Value {
	kind := dst.Kind()
	if kind == reflect.Slice || kind == reflect.Array {
		if dst.IsNil() {
			slice := reflect.MakeSlice(dst.Type(), 0, 1)
			dst.Set(slice)
		}
		if dst.Len() == 0 {
			var newType reflect.Type
			var isPointer bool
			if dst.Type().Elem().Kind() == reflect.Pointer {
				newType = dst.Type().Elem().Elem()
				isPointer = true
			} else {
				newType = dst.Type().Elem()
			}
			new := reflect.New(newType)
			if !isPointer {
				new = new.Elem()
			}
			dst.Set(reflect.Append(dst, new))
		}
		dst = dst.Index(0)
	}

	return dst
}
