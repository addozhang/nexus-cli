package nexus

import (
	"encoding/json"
	"time"
)

// TimeValue decodes Nexus timestamps, which use non-RFC3339 offsets such as
// "2018-01-01T00:00:00.000+0000". Nil pointers stay nil so callers can render
// explicit nulls.
type TimeValue time.Time

// Time returns the decoded time value.
func (t TimeValue) Time() time.Time { return time.Time(t) }

// UnmarshalJSON accepts the Nexus wire layouts plus standard RFC3339.
func (t *TimeValue) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		return nil
	}
	layouts := []string{
		"2006-01-02T15:04:05.999-0700",
		"2006-01-02T15:04:05.999Z07:00",
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, s); err == nil {
			*t = TimeValue(parsed)
			return nil
		}
	}
	return &time.ParseError{Value: s, Layout: layouts[0], Message: " (nexus timestamp)"}
}

// MarshalJSON renders RFC3339 for round-tripping.
func (t TimeValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Time().Format(time.RFC3339Nano))
}

// AsTime converts an optional wire timestamp pointer into a time.Time pointer.
// AsTime converts an optional wire timestamp pointer into a time.Time pointer.
func AsTime(tv *TimeValue) *time.Time {
	if tv == nil {
		return nil
	}
	t := tv.Time()
	return &t
}
