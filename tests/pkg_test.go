package pkg_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ilexPar/struct-marshal/pkg"
)

func TestStructureAnalysis(t *testing.T) {
	t.Run("should error when local is a pointer to a non struct value", func(t *testing.T) {
		wrongLocal := "asd"
		err := pkg.Introspect(&wrongLocal, APIObject{})
		assert.NotNil(t, err, "Expected error when destination pointed value is not a struct")
		assert.Equal(t, pkg.ErrLocalTypeNotStruct, err.Error())
		pkg.ClearTypeCache()
	})
	t.Run("should error when foreign is a pointer to a non struct value", func(t *testing.T) {
		wrongForeign := "asd"
		err := pkg.Introspect(SystemStruct{}, &wrongForeign)
		assert.NotNil(t, err, "Expected error when foreign pointed value is not a struct")
		assert.Equal(t, pkg.ErrForeignTypeNotStruct, err.Error())
		pkg.ClearTypeCache()
	})
	t.Run("should error if destination doesnt have any serializable field", func(t *testing.T) {
		invalidStruct := struct {
			Name   string
			Config string
		}{}
		err := pkg.Introspect(&invalidStruct, &APIObject{})
		assert.NotNil(t, err, "Expected error when destination is not a pointer")
		assert.Equal(t, pkg.ErrLocalTypeMissingValidTag, err.Error())
		pkg.ClearTypeCache()
	})
	t.Run("should error if foreign type does not have the pointed field by native type", func(t *testing.T) {
		native := struct {
			Name string `se:"Some"`
		}{}
		alien := struct{ NotSome string }{}
		err := pkg.Introspect(native, alien)
		assert.NotNil(t, err, "Expected error when destination is not a pointer")
		assert.Contains(t, err.Error(), pkg.ErrForeignTypeMissingField)
		pkg.ClearTypeCache()
	})
	t.Run("should error if native field points to a missmatching type", func(t *testing.T) {
		native := struct {
			Name string `se:"Some"`
		}{}
		alien := struct{ Some int }{}
		err := pkg.Introspect(native, alien)
		assert.NotNil(t, err, "Expected error when destination is not a pointer")
		assert.Contains(t, err.Error(), pkg.ErrForeignTypeMismatch)
		pkg.ClearTypeCache()
	})
	t.Run("should accept a list of types to assert", func(t *testing.T) {
		local := SystemStructWithMultipleDestination{}
		foreign1 := SecondaryAPIObject{Metadata: APIMetadata{}}
		foreign2 := APIObject{Metadata: APIMetadata{}}

		err1 := pkg.Introspect(local, foreign1)
		assert.Nil(t, err1)

		err2 := pkg.Introspect(local, foreign2)
		assert.Nil(t, err2)
		pkg.ClearTypeCache()
	})
	t.Run("should accept per-type path when using type matching", func(t *testing.T) {
		local := SystemStructWithMultipleDestination{}
		foreign1 := SecondaryAPIObject{}
		foreign2 := APIObject{}

		err1 := pkg.Introspect(local, foreign1)
		err2 := pkg.Introspect(local, foreign2)

		assert.Nil(t, err1)
		assert.Nil(t, err2)
		pkg.ClearTypeCache()
	})
	t.Run("should error when using per-type path matching and main path is not right operator", func(t *testing.T) {
		type LocalStruct struct {
			Flag bool `se:"metadata.flag,types<APIObject:metadata.flag|SecondaryAPIObject:configflag>"`
		}

		err := pkg.Introspect(LocalStruct{}, APIObject{})
		assert.ErrorContains(t, err, pkg.ErrInvalidPerTypePath)
		pkg.ClearTypeCache()
	})
}

func TestUnmarshal(t *testing.T) {
	name := "test"
	count := 999
	flag := true
	direction := "up"
	list := []string{"a", "b", "c"}
	t.Run("should error when destination is not a pointer", func(t *testing.T) {
		var emptyDst SystemStruct
		err := pkg.Unmarshal(APIObject{}, emptyDst)
		assert.NotNil(t, err, "Expected error when destination is not a pointer")
		assert.Equal(t, pkg.ErrUnmarshalDestType, err.Error())
		pkg.ClearTypeCache()
	})
	t.Run("should error when destination is nil", func(t *testing.T) {
		var emptyDst *SystemStruct
		err := pkg.Unmarshal(APIObject{}, emptyDst)
		assert.NotNil(t, err, "Expected error when destination pointed value is nil")
		assert.Equal(t, pkg.ErrUnmarshalDestType, err.Error())
		pkg.ClearTypeCache()
	})
	t.Run("should accept a struct value as source", func(t *testing.T) {
		err := pkg.Unmarshal(APIObject{}, &SystemStruct{})
		assert.Nil(t, err, "Expected no error when destination is a struct value")
		pkg.ClearTypeCache()
	})
	t.Run("should error when destination is a pointer to a non struct value", func(t *testing.T) {
		badDst := name
		err := pkg.Unmarshal(APIObject{}, &badDst)
		assert.NotNil(t, err, "Expected error when destination pointed value is not a struct")
		assert.Equal(t, pkg.ErrUnmarshalDestType, err.Error())
		pkg.ClearTypeCache()
	})
	t.Run("should error when source is a pointer to a non struct value", func(t *testing.T) {
		badSrc := name
		err := pkg.Unmarshal(&badSrc, &SystemStruct{})
		assert.NotNil(t, err, "Expected error when source pointed value is not a struct")
		assert.Equal(t, pkg.ErrUnmarshalSrcType, err.Error())
		pkg.ClearTypeCache()
	})
	t.Run("should accept a pointer to a non nil struct as source", func(t *testing.T) {
		err := pkg.Unmarshal(&APIObject{}, &SystemStruct{})
		assert.Nil(t, err, "Expected no error when destination is a pointer to a struct")
		pkg.ClearTypeCache()
	})
	t.Run("should error if source pointer value is nil", func(t *testing.T) {
		var emptySrc *APIObject
		err := pkg.Unmarshal(emptySrc, &SystemStruct{})
		assert.NotNil(t, err, "Expected error when destination pointed value is nil")
		assert.Equal(t, pkg.ErrUnmarshalSrcType, err.Error())
		pkg.ClearTypeCache()
	})
	t.Run("correct unmarshal", func(t *testing.T) {
		dst := SystemStruct{}

		src := APIObject{
			Metadata: APIMetadata{
				NameField: name,
				Flag:      flag,
			},
			Config: APIConfig{
				SomeCount: count,
				SomeList: []APIListedObj{
					{
						List: list,
						Config: APIListedObjConfig{
							Direction: direction,
							DeepNested: APIDeepNested{
								Direction2: direction,
							},
						},
					},
				},
				SomeList2: []*APIListedObj{
					{
						Config: APIListedObjConfig{
							Direction: direction,
						},
					},
				},
			},
		}

		err := pkg.Unmarshal(src, &dst)

		assert.Nil(t, err)
		assert.Equal(t, name, dst.Name)
		assert.Equal(t, count, dst.Count)
		assert.Equal(t, flag, dst.Flag)
		assert.Equal(t, list, dst.ListedStuff)
		assert.Equal(t, direction, dst.Nested.Direction)
		assert.Equal(t, direction, dst.Nested.DeeepNested.Direction)
		assert.Equal(t, direction, dst.NestedPointer.Direction)
		assert.Equal(t, direction, dst.StructSlice[0].Direction)
		pkg.ClearTypeCache()
	})
	t.Run("should ignore fields that doesn't match type matching option", func(t *testing.T) {
		dst := struct {
			Name string `se:"Metadata.NameField,types<SecondaryAPIObject>"`
			Flag bool   `se:"ConfigFlag,types<APIObject>"`
		}{}
		src := SecondaryAPIObject{
			Metadata: APIMetadata{
				NameField: "test",
			},
			ConfigFlag: true,
		}

		err := pkg.Unmarshal(src, &dst)

		assert.Nil(t, err)
		assert.False(t, dst.Flag)
		pkg.ClearTypeCache()
	})

	//
	t.Run("should use the full path of nested fields when using operator", func(t *testing.T) {
		dst := &SystemStructWithMultipleDestination{}
		src := SecondaryAPIObject{
			Child: SecondaryAPIObjectChild{
				Direction: "up",
			},
		}

		err := pkg.Unmarshal(src, dst)
		assert.Nil(t, err)
		assert.Equal(t, src.Child.Direction, dst.DismissNested.Direction)
		pkg.ClearTypeCache()
	})
	t.Run("should skip processing empty slice source values", func(t *testing.T) {
		dst := SystemStruct{}
		src := APIObject{
			Metadata: APIMetadata{
				NameField: name,
			},
			Config: APIConfig{},
		}

		err := pkg.Unmarshal(src, &dst)

		assert.Nil(t, err)
		assert.Equal(t, name, dst.Name)
		pkg.ClearTypeCache()
	})
	t.Run("should update data if destination has values", func(t *testing.T) {
		dontReplace := "should-not-replace"
		nestedDirection := "down"
		list := []string{"valid", "list"}
		dst := &SystemStruct{
			Name:        "should-replace",
			ListedStuff: []string{"non", "valid"},
			Nested: SystemNested{
				Direction: "should-replace",
				DeeepNested: SystemDeepNested{
					Direction: "should-replace",
				},
			},
			NestedPointer: &SystemNested{
				Direction: dontReplace,
				DeeepNested: SystemDeepNested{
					Direction: "should-replace",
				},
			},
			StructSlice: []SystemNestedFromSlice{
				{Direction: "should-replace"},
			},
		}

		src := APIObject{
			Metadata: APIMetadata{
				NameField: name,
			},
			Config: APIConfig{
				SomeList: []APIListedObj{
					{
						List: list,
						Config: APIListedObjConfig{
							Direction: direction,
							DeepNested: APIDeepNested{
								Direction2: nestedDirection,
							},
						},
					},
				},
				SomeList2: []*APIListedObj{
					{
						Config: APIListedObjConfig{
							DeepNested: APIDeepNested{
								Direction2: nestedDirection,
							},
						},
					},
				},
			},
		}

		err := pkg.Unmarshal(src, dst)

		assert.Nil(t, err)
		assert.Equal(t, name, dst.Name)
		assert.Equal(t, dontReplace, dst.NestedPointer.Direction)
		pkg.ClearTypeCache()
	})
}

func TestMarshal(t *testing.T) {
	name := "test"
	count := 999
	flag := true
	direction := "up"
	list := []string{"a", "b", "c"}
	src := SystemStruct{
		Name:  name,
		Count: count,
		Flag:  flag,
		Nested: SystemNested{
			Direction: direction,
		},
		NestedPointer: &SystemNested{
			Direction: direction,
		},
		ListedStuff: list,
	}

	t.Run("should error when destination is not a pointer", func(t *testing.T) {
		var emptyDst APIObject
		err := pkg.Marshal(SystemStruct{}, emptyDst)
		assert.NotNil(t, err, "Expected error when destination is not a pointer")
		assert.Equal(t, pkg.ErrUnmarshalDestType, err.Error())
		pkg.ClearTypeCache()
	})
	t.Run("should error when destination is nil", func(t *testing.T) {
		var emptyDst *APIObject
		err := pkg.Marshal(SystemStruct{}, emptyDst)
		assert.NotNil(t, err, "Expected error when destination pointed value is nil")
		assert.Equal(t, pkg.ErrUnmarshalDestType, err.Error())
		pkg.ClearTypeCache()
	})
	t.Run("should accept a struct value as source", func(t *testing.T) {
		err := pkg.Marshal(SystemStruct{}, &APIObject{})
		assert.Nil(t, err, "Expected no error when destination is a struct value")
		pkg.ClearTypeCache()
	})
	t.Run("should error when destination is a pointer to a non struct value", func(t *testing.T) {
		emptyDst := name
		err := pkg.Marshal(APIObject{}, &emptyDst)
		assert.NotNil(t, err, "Expected error when destination pointed value is not a struct")
		assert.Equal(t, pkg.ErrUnmarshalDestType, err.Error())
		pkg.ClearTypeCache()
	})
	t.Run("should error when source is a pointer to a non struct value", func(t *testing.T) {
		badSrc := name
		err := pkg.Marshal(&badSrc, &APIObject{})
		assert.NotNil(t, err, "Expected error when source pointed value is not a struct")
		assert.Equal(t, pkg.ErrUnmarshalSrcType, err.Error())
		pkg.ClearTypeCache()
	})
	t.Run("should accept a pointer to a non nil struct as source", func(t *testing.T) {
		err := pkg.Marshal(&SystemStruct{}, &APIObject{})
		assert.Nil(t, err, "Expected no error when destination is a pointer to a struct")
		pkg.ClearTypeCache()
	})
	t.Run("should error if source pointer value is nil", func(t *testing.T) {
		var emptySrc *APIObject
		err := pkg.Marshal(emptySrc, &SystemStruct{})
		assert.NotNil(t, err, "Expected error when destination pointed value is nil")
		assert.Equal(t, pkg.ErrUnmarshalSrcType, err.Error())
		pkg.ClearTypeCache()
	})
	t.Run("should marshal source into destination", func(t *testing.T) {
		dst := &APIObject{}
		err := pkg.Marshal(src, dst)

		assert.Nil(t, err)
		assert.Equal(t, name, dst.Metadata.NameField)
		assert.Equal(t, flag, dst.Metadata.Flag)
		assert.Equal(t, count, dst.Config.SomeCount)
		assert.Equal(t, list, dst.Config.SomeList[0].List)
		assert.Equal(t, direction, dst.Config.SomeList[0].Config.Direction)
		assert.Equal(t, direction, dst.Config.SomeList2[0].Config.Direction)
		pkg.ClearTypeCache()
	})
}
