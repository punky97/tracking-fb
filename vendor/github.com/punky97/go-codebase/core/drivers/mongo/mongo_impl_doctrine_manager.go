package mongo

var doctrineManager = struct{ Collection CollectionInfo }{
	Collection: CollectionInfo{
		Name: "doctrine_increment_ids",
		ConvertToSnakeCaseField: []string{"currentId"},
	},
}
