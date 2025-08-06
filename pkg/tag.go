package pkg

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
)

type TypeMatch struct {
	Path    []string
	Name    string
	Matches bool
}

type TagOpts struct {
	MatchTypes []TypeMatch
}

type FieldTag struct {
	Path       []string
	Opts       TagOpts
	Skip       bool
	TargetType string
}

// check naming convention when using "type matching" tag option
// return values can either be:
// - TypeMatch with `Matches` property set true if no type-matching options set in this tag field
// - TypeMatch with `Matches` property set false if no match is found but there are type-matching options set in this
// tag field
// - the TypeMatch description if a match is found
func (t *FieldTag) findTypeMatch(typeName string) TypeMatch {
	result := TypeMatch{Matches: true}
	if len(t.Opts.MatchTypes) > 0 && typeName != "" {
		result.Matches = false
		for _, match := range t.Opts.MatchTypes {
			if match.Name == typeName {
				result = match
				result.Matches = true
				break
			}
		}
	}
	return result
}

// validatePaths checks if the paths in a FieldTag and a TypeMatch are valid together.
// It returns an error if:
// 1. The TypeMatch has a path AND the FieldTag's first path element is the a multi-type operator
// 2. The TypeMatch has a path AND the FieldTag's path has more than one element
//
// This validation ensures that type-specific paths are used correctly and don't
// conflict with the main path definition.
func (this *FieldTag) validatePaths(match TypeMatch) error {
	var err error
	var failedCheck bool

	matchHasPath := len(match.Path) > 0
	mainPathMatches := this.Path[0] != MULTI_TYPE_NAME
	mainPathMaxLen := len(this.Path) > 1

	if matchHasPath && mainPathMatches {
		failedCheck = true
	}

	if matchHasPath && mainPathMaxLen {
		failedCheck = true
	}

	if failedCheck {
		err = errors.New(ErrInvalidPerTypePath)
	}

	return err
}

// validate checks if a FieldTag is valid for the given foreign root type.
// It returns an error if the tag is invalid, or nil if the tag is valid.
//
// If the tag has Skip set to true, the function returns nil.
//
// The function first checks if the foreign root type matches any of the type
// matches specified in the tag options. If no match is found, the tag is marked
// as Skip and the function returns nil.
//
// If a match is found, the function checks if the per-type path naming is valid
// using checkPerTypePathNaming. If the match has a path, the function replaces
// the tag's main path with the match's path.
func (this *FieldTag) validate(foreignRootType string) error {
	if this.Skip {
		return nil
	}

	match := this.findTypeMatch(foreignRootType)
	if !match.Matches {
		this.Skip = true
		return nil
	}

	err := this.validatePaths(match)

	if len(match.Path) > 0 {
		// replace tag main path with type-matching path
		this.Path = match.Path
	}

	return err
}

// getTagAndTarget processes a struct field and returns a FieldTag, the target field ref key, and any error encountered.
//
// Parameters:
// - foreignRoot: The name of the foreign root type to validate against
// - field: The reflect.StructField being processed
// - alien: The reflect.Type of the foreign struct being matched against
// - parentPath: The path from parent fields, if any
//
// Returns:
// - FieldTag: The parsed and validated field tag
// - string: The target field name in the foreign struct
// - error: Any error encountered during processing
//
// The function first parses the field's tag, then validates it against the foreign root type.
// If the tag is set to be skipped or validation fails, it returns early.
// Otherwise, it processes the path (handling nested fields appropriately) and
// determines the target field name and type in the foreign struct.
func getTagAndTarget(
	foreignRoot string,
	field reflect.StructField,
	alien reflect.Type,
	parentPath []string,
) (FieldTag, string, error) {
	var err error
	tag := parseTag(field)
	err = tag.validate(foreignRoot)
	if tag.Skip || err != nil {
		return tag, "", err
	}

	var target, targetType string
	if tag.Path[0] == DISMISS_NESTED {
		tag.Path = parentPath
	} else {
		if len(parentPath) > 0 {
			tag.Path = append(parentPath, tag.Path...)
		}
		if target, targetType, err = parseTargetField(tag.Path, alien); err != nil {
			return tag, "", err
		}
	}

	tag.TargetType = targetType

	return tag, target, err
}

// parseTag parses a field tag string into a FieldTag struct. The field tag string
// is expected to be in the format "path,opt1,opt2,...". The path is split on
// periods to create the Path field of the FieldTag struct. The remaining comma-
// separated values are parsed into the Opts field of the FieldTag struct.
//
// If the field tag string is empty, the function returns a FieldTag with skip
// set to true.
func parseTag(field reflect.StructField) FieldTag {
	tag := FieldTag{}
	rawString := field.Tag.Get(FIELD_TAG_KEY)
	if rawString == "" {
		tag.Skip = true
		return tag
	}

	tagParts := strings.Split(rawString, ",")
	tag.Path = strings.Split(tagParts[0], ".")

	if len(tagParts) > 1 {
		tag.Opts = parseTagOpts(tagParts[1:])
	}

	return tag
}

// parseTagOpts parses a list of tag options into a TagOpts struct.
// The options are expected to be in the format "opt1,opt2,...".
// The resulting TagOpts will contain a list of TypeMatch structs, one for each type option.
func parseTagOpts(opts []string) TagOpts {
	options := TagOpts{}
	matchTypeRegEx := regexp.MustCompile(TYPE_OPTS_REGEX)
	for _, opt := range opts {
		typeMatches := matchTypeRegEx.FindStringSubmatch(opt)
		if len(typeMatches) > 0 {
			parseTypeMatches(typeMatches[1], &options.MatchTypes)
		}
	}
	return options
}

// parseTypeMatches parses a string representation of type matches into a slice of TypeMatch structs.
// The input string is expected to be in the format "typeName1:fieldPath1|typeName2:fieldPath2|...".
// Each type match consists of a type name and an optional field path, separated by a colon.
// The field paths are split on periods to create the Path field of the TypeMatch struct.
// The resulting slice contains one TypeMatch struct for each type match in the input string.
func parseTypeMatches(data string, matches *[]TypeMatch) {
	parts := strings.Split(data, "|")
	for _, typeOpt := range parts {
		var fieldPath []string
		typeParts := strings.Split(typeOpt, ":")
		typeName := typeParts[0]
		if len(typeParts) > 1 {
			fieldPath = strings.Split(typeParts[1], ".")
		}
		*matches = append(*matches, TypeMatch{
			Name: typeName,
			Path: fieldPath,
		})
	}
}

// parseTargetField parses a path in the target structure to locate a specific field and generates
// a unique reference key for that field. It traverses the target structure recursively following
// the path components to find the desired field.
//
// Parameters:
//   - path: A slice of strings representing the path to the target field (e.g., ["Person", "Address", "Street"]).
//   - foreign: The reflect.Type of the target (foreign) structure to search within.
//   - fullPath: An optional parameter tracking the full path traversed so far, used in recursive calls.
//
// Returns:
//   - string: A unique key for the target field that can be used to reference it in the foreignRepresentations map.
//   - string: The name of the field's type.
//   - error: An error if the field cannot be found or if there's an issue during traversal.
//
// The function handles various field types including nested structs, arrays, maps, and pointers.
// It builds both a string path representation and an index path that can be used for direct
// field access via reflection. The results are stored in the foreignRepresentations map
// for later use during the mapping process.
func parseTargetField(path []string, foreign reflect.Type, fullPath ...[]interface{}) (string, string, error) {
	descendableFields := []reflect.Kind{reflect.Map, reflect.Array, reflect.Slice}
	arrayReg := regexp.MustCompile(`^([a-zA-Z0-9_]+)\[0\]$`)
	if len(path) == 0 {
		return "", "", errors.New("empty tag path")
	}
	if len(fullPath) == 0 {
		fullPath = [][]interface{}{}
	}

	for id := range foreign.NumField() {
		field := foreign.Field(id)
		fieldType := field.Type
		fieldKind := fieldType.Kind()
		pathName := path[0]

		arrayMatch := arrayReg.FindStringSubmatch(path[0])
		if len(arrayMatch) > 1 {
			pathName = arrayMatch[1]
		}
		if slices.Contains(descendableFields, fieldKind) {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}

		if field.Name == pathName {
			return extractTargetData(id, pathName, path, field, foreign, fieldType, fullPath...)
		}
	}

	return "", "", fmt.Errorf(ErrForeignTypeMissingField+" %v", path)
}

func extractTargetData(
	id int,
	pathName string,
	path []string,
	field reflect.StructField,
	foreign, fieldType reflect.Type,
	fullPath ...[]interface{},
) (string, string, error) {
	if len(path) == 1 {
		namedPath := []string{}
		indexPath := []int{}
		for i := range fullPath {
			p, ok := fullPath[i][0].(int)
			if !ok {
				return "", "", fmt.Errorf("ERROR: Somehow we messed up with index type inference")
			}
			indexPath = append(indexPath, p)
			namedPath = append(namedPath, fmt.Sprintf("%v", fullPath[i][1]))
		}
		key := getForeignTargetKey(foreign, field.Name, namedPath)
		namedPath = append(namedPath, pathName)
		indexPath = append(indexPath, id)
		// TODO: maybe namedPath is not needed at all
		foreignRepresentations[key] = TargetField{
			Id:        id,
			Path:      namedPath,
			Kind:      fieldType.Kind(),
			IndexPath: indexPath,
		}
		return key, fieldType.Name(), nil
	}
	fullPath = append(fullPath, []interface{}{id, pathName})
	return parseTargetField(path[1:], fieldType, fullPath...)
}
