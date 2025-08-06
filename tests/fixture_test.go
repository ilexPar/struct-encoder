package pkg_test

// Mock a struct internal to an application
type SystemDeepNested struct {
	Direction string `se:"Direction2"`
}
type SystemNested struct {
	Direction   string           `se:"Direction"`
	DeeepNested SystemDeepNested `se:"DeepNested"`
}
type SystemNestedFromSlice struct {
	Direction string `se:"Config.Direction"`
}
type SystemStruct struct {
	Name          string                  `se:"Metadata.NameField"`
	Count         int                     `se:"Config.SomeCount"`
	Flag          bool                    `se:"Metadata.Flag"`
	Nested        SystemNested            `se:"Config.SomeList[0].Config"`
	NestedPointer *SystemNested           `se:"Config.SomeList2[0].Config"`
	ListedStuff   []string                `se:"Config.SomeList[0].List"`
	StructSlice   []SystemNestedFromSlice `se:"Config.SomeList"`
}

// Mock a struct taking advantage of multiple destination types
type SystemStructWithMultipleDestination struct {
	Name                 string                                `se:"Metadata.NameField,types<APIObject|SecondaryAPIObject>"`         // multiple type checks using unified path
	Flag                 bool                                  `se:"+,types<APIObject:Metadata.Flag|SecondaryAPIObject:ConfigFlag>"` // multiple type checks using per type path
	Blackhole            string                                `se:"dismiss,types<InexistentType>"`                                  // this should be ignored
	DismissNested        NestedStructWithMultipleDestinations  `se:"->"`                                                             // let the nested fields declare the full destination path
	DismissNestedPointer *NestedStructWithMultipleDestinations `se:"->"`
}
type NestedStructWithMultipleDestinations struct {
	Direction string `se:"Child.Direction,types<SecondaryAPIObject>"`
}

// Mock a struct that differs in structure from our internal struct, probably belonging to another API
type APIObject struct {
	Metadata APIMetadata
	Config   APIConfig
}
type APIDeepNested struct {
	Direction2 string
}
type APIListedObjConfig struct {
	DeepNested APIDeepNested
	Direction  string
}
type APIListedObj struct {
	List   []string
	Config APIListedObjConfig
}
type APIMetadata struct {
	NameField string
	Flag      bool
}
type APIConfig struct {
	SomeCount   int
	SomeList    []APIListedObj
	SomeList2   []*APIListedObj
	SomePointed *APIListedObj
}

// Mock another struct that differs in structure from both our internal struct and the APIObject
// to test multiple types compatibility
type SecondaryAPIObjectChild struct {
	Direction string
}
type SecondaryAPIObject struct {
	Metadata   APIMetadata
	ConfigFlag bool
	Child      SecondaryAPIObjectChild
}
