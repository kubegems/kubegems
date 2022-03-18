package set

import (
	"testing"

	"kubegems.io/pkg/utils/slice"
)

func TestSlice(t *testing.T) {
	tests := []struct {
		name       string
		elems      []string
		want       []string
		wantlength int
	}{
		{
			name:       "string",
			elems:      []string{"a", "a", "b", "c"},
			want:       []string{"a", "b", "c"},
			wantlength: 3,
		},
	}

	for _, tt := range tests {
		set := NewSet[string]()
		set.Append(tt.elems...)
		t.Run(tt.name, func(t *testing.T) {
			if got := set.Slice(); slice.SliceUniqueKey(got) != slice.SliceUniqueKey(tt.want) {
				t.Errorf("Slice() = %v, want %v", got, tt.want)
			}
			if length := set.Len(); length != tt.wantlength {
				t.Errorf("Len() = %v, want %v", length, tt.wantlength)
			}
		})
	}
}
