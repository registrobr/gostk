package path

import "testing"

func TestRelevantPath(t *testing.T) {
	scenarios := []struct {
		description string
		path        string
		n           int
		expected    string
	}{
		{
			description: "it should remove the extra directories correctly",
			path:        "1/2/3/4/5",
			n:           3,
			expected:    "3/4/5",
		},
		{
			description: "it should avoid removing directories when n is bigger than the number of directories",
			path:        "1/2/3/4/5",
			n:           5,
			expected:    "1/2/3/4/5",
		},
		{
			description: "it should avoid removing directories when n is zero",
			path:        "1/2/3/4/5",
			n:           0,
			expected:    "1/2/3/4/5",
		},
		{
			description: "it should avoid removing directories when n is negative",
			path:        "1/2/3/4/5",
			n:           -1,
			expected:    "1/2/3/4/5",
		},
	}

	for i, scenario := range scenarios {
		path := RelevantPath(scenario.path, scenario.n)
		if scenario.expected != path {
			t.Errorf("scenario %d, “%s”: mismatch results. Expecting: “%v”; found “%v”",
				i, scenario.description, scenario.expected, path)
		}
	}
}
