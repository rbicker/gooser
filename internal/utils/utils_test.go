package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsMailAddress tests the IsMailAddress function.
func TestIsMailAddress(t *testing.T) {
	tests := []struct {
		mail string
		want bool
	}{
		{
			mail: "test@example.com",
			want: true,
		},
		{
			mail: "test@example_abc.com",
			want: false,
		},
		{
			mail: "test@example-abc.com",
			want: true,
		},
		{
			mail: "test@example",
			want: true,
		},
		{
			mail: "test@example@example.com",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.mail, func(t *testing.T) {
			if got := IsMailAddress(tt.mail); got != tt.want {
				t.Errorf("IsMailAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppendUniqueString(t *testing.T) {
	tests := []struct {
		name        string
		ss          []string
		s           string
		wantSlice   []string
		wantChanged bool
	}{
		{
			name:        "simple",
			ss:          []string{"x", "y"},
			s:           "z",
			wantSlice:   []string{"x", "y", "z"},
			wantChanged: true,
		},
		{
			name:        "no change",
			ss:          []string{"x", "y"},
			s:           "y",
			wantSlice:   []string{"x", "y"},
			wantChanged: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			gotSlice, gotChanged := AppendUniqueString(tt.ss, tt.s)
			assert.Equal(tt.wantSlice, gotSlice)
			assert.Equal(tt.wantChanged, gotChanged)
		})
	}
}

func TestRandomString(t *testing.T) {
	tests := []struct {
		name       string
		n          int
		wantLength int
	}{
		{
			name:       "0",
			n:          0,
			wantLength: 0,
		},
		{
			name:       "1",
			n:          1,
			wantLength: 1,
		},
		{
			name:       "5",
			n:          5,
			wantLength: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			got := RandomString(tt.n)
			assert.Equal(tt.wantLength, len(got))
		})
	}
}

func TestRemoveFromStringSlice(t *testing.T) {
	tests := []struct {
		name string
		ss   []string
		s    string
		want []string
	}{
		{
			name: "simple",
			ss:   []string{"a", "b", "c"},
			s:    "a",
			want: []string{"b", "c"},
		},
		{
			name: "non-existing",
			ss:   []string{"a", "b", "c"},
			s:    "x",
			want: []string{"a", "b", "c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			got := RemoveFromStringSlice(tt.ss, tt.s)
			assert.ElementsMatch(tt.want, got)
		})
	}
}

func TestStringSlicesDiff(t *testing.T) {
	tests := []struct {
		name        string
		a           []string
		b           []string
		wantAdded   []string
		wantRemoved []string
	}{
		{
			name:      "adding",
			a:         []string{"a", "b"},
			b:         []string{"a", "b", "c"},
			wantAdded: []string{"c"},
		},
		{
			name:        "removing",
			a:           []string{"a", "b", "c"},
			b:           []string{"a", "b"},
			wantRemoved: []string{"c"},
		},
		{
			name:        "both",
			a:           []string{"a", "b", "c", "z"},
			b:           []string{"a", "b", "c", "x", "y"},
			wantAdded:   []string{"x", "y"},
			wantRemoved: []string{"z"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			gotAdded, gotRemoved := StringSlicesDiff(tt.a, tt.b)
			assert.ElementsMatch(tt.wantAdded, gotAdded, "added does not match")
			assert.ElementsMatch(tt.wantRemoved, gotRemoved, "removed does not match")
		})
	}
}

func TestUniqueStringSlice(t *testing.T) {

	tests := []struct {
		name        string
		ss          []string
		wantSlice   []string
		wantChanged bool
	}{
		{
			name:        "already unique",
			ss:          []string{"a", "b", "c"},
			wantSlice:   []string{"a", "b", "c"},
			wantChanged: false,
		},
		{
			name:        "not unique",
			ss:          []string{"a", "c", "b", "a", "c"},
			wantSlice:   []string{"a", "b", "c"},
			wantChanged: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			gotSlice, gotChanged := UniqueStringSlice(tt.ss)
			assert.ElementsMatch(tt.wantSlice, gotSlice, "slice does not match")
			assert.Equal(tt.wantChanged, gotChanged, "changed bool does not match")

		})
	}
}
