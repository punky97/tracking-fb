package dto

type Pixel struct {
	Id       string `json:"id"`
	Token    string `json:"token"`
	TestCode string `json:"test_code"`
}

type Data struct {
	Data          []*DataItem `json:"data"`
	TestEventCode string      `json:"test_event_code"`
}

type DataItem struct {
	EventName      string                 `json:"event_name"`
	EventTime      int64                  `json:"event_time"`
	EventId        string                 `json:"event_id"`
	EventSourceUrl string                 `json:"event_source_url"`
	ActionSource   string                 `json:"action_source"`
	UserData       map[string]interface{} `json:"user_data"`
	CustomData     map[string]interface{} `json:"custom_data"`
	OptOut         bool                   `json:"opt_out"`
}
