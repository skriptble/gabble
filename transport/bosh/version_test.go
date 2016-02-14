package bosh

import "testing"

func TestVersion(t *testing.T) {
	t.Parallel()
	var tests = []struct {
		first, second, ret Version
		err                string
	}{
		{
			first:  Version{Major: 0, Minor: 1},
			second: Version{Major: 1, Minor: 0},
			ret:    Version{Major: 1, Minor: 0},
			err:    "Should return second when second has higher major",
		},
		{
			first:  Version{Major: 1, Minor: 1},
			second: Version{Major: 0, Minor: 2},
			ret:    Version{Major: 1, Minor: 1},
			err:    "Should return first when first has higher major",
		},
		{
			first:  Version{Major: 2, Minor: 3},
			second: Version{Major: 2, Minor: 4},
			ret:    Version{Major: 2, Minor: 4},
			err:    "Should return second when second has higher minor",
		},
		{
			first:  Version{Major: 3, Minor: 4},
			second: Version{Major: 3, Minor: 3},
			ret:    Version{Major: 3, Minor: 4},
			err:    "Should return first when first has higher minor",
		},
		{
			first:  Version{Major: 4, Minor: 5},
			second: Version{Major: 4, Minor: 5},
			ret:    Version{Major: 4, Minor: 5},
			err:    "Should return first or second when major and minor match",
		},
	}
	var want, got Version
	for _, test := range tests {
		want = test.ret
		got = test.first.Compare(test.second)
		if want != got {
			t.Error(test.err)
			t.Errorf("\nWant:%+v\nGot :%+v", want, got)
		}
	}
}
