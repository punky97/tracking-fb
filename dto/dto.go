package dto

type Action struct {
	Uuid       string
	SessionId  string
	EventId    int64
	EventType  string
	CustomerId int64
	Params     map[string]string
	CreatedAt  int64
	UpdatedAt  int64
	Sources    []string
	Code       string
}
