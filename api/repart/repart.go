package repart

// Output is the structured output of systemd-repart.
type Output []Partition

// Partition is a partition in the stuctured output of systemd-repart.
type Partition struct {
	Type       string `json:"type"`
	Label      string `json:"label"`
	UUID       string `json:"uuid"`
	Partno     int64  `json:"partno"`
	File       string `json:"file"`
	Node       string `json:"node"`
	Offset     int64  `json:"offset"`
	OldSize    int64  `json:"old_size"`
	RawSize    int64  `json:"raw_size"`
	OldPadding int64  `json:"old_padding"`
	RawPadding int64  `json:"raw_padding"`
	Activity   string `json:"activity"`
	Roothash   string `json:"roothash,omitempty"`
	Usrhash    string `json:"usrhash,omitempty"`
}
