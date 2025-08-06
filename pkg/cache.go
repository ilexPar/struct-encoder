package pkg

var localRepresentations map[string]StructRepr

var foreignRepresentations map[string]TargetField

func cacheInit() {
	if localRepresentations == nil {
		localRepresentations = map[string]StructRepr{}
	}
	if foreignRepresentations == nil {
		foreignRepresentations = map[string]TargetField{}
	}
}

// ClearTypeCache empties the internal cache of type representations,
// resetting both localRepresentations and foreignRepresentations maps to empty maps.
// This can be useful when the type information needs to be refreshed or when
// freeing up memory in long-running applications.
func ClearTypeCache() {
	localRepresentations = map[string]StructRepr{}
	foreignRepresentations = map[string]TargetField{}
}
