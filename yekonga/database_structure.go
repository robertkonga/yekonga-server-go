package yekonga

type CollectionStructure struct {
	Name   string
	Fields map[string]CollectionStructureField
}

type CollectionStructureField struct {
	PrimaryKey   bool
	Name         string
	Kind         string
	DefaultValue interface{}
	Required     bool
	ForeignKey   CollectionStructureFieldForeignKey
}

type CollectionStructureFieldForeignKey struct {
	Model string
	Key   string
}
