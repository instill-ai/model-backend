package datamodel

import (
	"testing"

	"github.com/frankban/quicktest"
)

func TestDatamodel_TagNames(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		model    *Model
		expected []string
	}{
		{
			model: &Model{
				Tags: []*ModelTag{
					{
						TagName: "tag1",
					},
					{
						TagName: "tag2",
					},
				},
			},
			expected: []string{"tag1", "tag2"},
		},
		{
			model:    &Model{},
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		tagNames := tc.model.TagNames()
		c.Assert(tagNames, quicktest.DeepEquals, tc.expected)
	}
}
