package types

type RegistryEvent struct {
	Events []Event `json:"events"`
}

type Event struct {
	ID        string `json:"id"`
	Action    string `json:"action"`
	Target    Target `json:"target"`
	Timestamp string `json:"timestamp"`
	Request   struct {
		ID        string `json:"id"`
		Addr      string `json:"addr"`
		Host      string `json:"host"`
		Method    string `json:"method"`
		UserAgent string `json:"useragent"`
	} `json:"request"`
	Actor struct {
		Name string `json:"name"`
	} `json:"actor"`
}

type Target struct {
	Digest     string `json:"digest"`
	Repository string `json:"repository"`
	Tag        string `json:"tag,omitempty"`
}
