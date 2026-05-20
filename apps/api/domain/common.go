package domain

import (
	"fmt"
	"io"
)

type UInt32 uint32
type MAP map[string]any

func (u UInt32) MarshalGQL(w io.Writer) {
	fmt.Fprintf(w, "%d", u)
}

func (u *UInt32) UnmarshalGQL(v any) error {
	switch val := v.(type) {
	case int64:
		*u = UInt32(val)
	case float64:
		*u = UInt32(val)
	case string:
		var i uint32
		if _, err := fmt.Sscanf(val, "%d", &i); err != nil {
			return fmt.Errorf("UInt32 must be a number: %w", err)
		}
		*u = UInt32(i)
	default:
		return fmt.Errorf("UInt32 cannot be %T", v)
	}
	return nil
}
