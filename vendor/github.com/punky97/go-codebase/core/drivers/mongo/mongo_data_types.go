package mongo

// Paginate -- The struct for pagination request
type Paginate struct {
	Limit  int
	Offset int
}

func GetDefaultPaginate() Paginate {
	return Paginate{Limit: 10, Offset: 0}
}
