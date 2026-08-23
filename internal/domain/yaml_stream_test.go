package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestYAMLStreamDocuments(t *testing.T) {
	tests := []struct {
		name              string
		contents          string
		expectedPositions []int
		expectedDocuments []string
	}{
		{
			name:              "SingleDocument_Whole",
			contents:          "kind: Installation\n",
			expectedPositions: []int{1},
			expectedDocuments: []string{"kind: Installation\n"},
		},
		{
			name:              "TwoDocuments_Split",
			contents:          "kind: Installation\n---\nkind: CollectionAgent\n",
			expectedPositions: []int{1, 2},
			expectedDocuments: []string{"kind: Installation\n", "kind: CollectionAgent\n"},
		},
		{
			// A separator opening the stream stays with the document it opens,
			// where it is a document-start marker and still parses.
			name:              "LeadingSeparator_KeptWithFirstDocument",
			contents:          "---\nkind: Installation\n",
			expectedPositions: []int{1},
			expectedDocuments: []string{"---\nkind: Installation\n"},
		},
		{
			name:              "TrailingSeparator_NoEmptyDocument",
			contents:          "kind: Installation\n---\n",
			expectedPositions: []int{1},
			expectedDocuments: []string{"kind: Installation\n"},
		},
		{
			name:              "SeparatorWithComment_Split",
			contents:          "kind: Installation\n--- # the agent\nkind: CollectionAgent\n",
			expectedPositions: []int{1, 2},
			expectedDocuments: []string{"kind: Installation\n", "kind: CollectionAgent\n"},
		},
		{
			// The skipped documents keep their positions, so the third document
			// of the file is reported as the third.
			name:              "BlankAndCommentDocuments_SkippedButCounted",
			contents:          "# a note\n---\n\n---\nkind: Installation\n",
			expectedPositions: []int{3},
			expectedDocuments: []string{"kind: Installation\n"},
		},
		{
			name:              "WindowsLineEndings_Normalized",
			contents:          "kind: Installation\r\n---\r\nkind: CollectionAgent\r\n",
			expectedPositions: []int{1, 2},
			expectedDocuments: []string{"kind: Installation\n", "kind: CollectionAgent\n"},
		},
		{
			// A separator inside a block scalar belongs to the config the
			// scalar carries, not to the stream.
			name:              "SeparatorInsideBlockScalar_NotSplit",
			contents:          "config:\n  data: |\n    ---\n    receivers: [otlp]\n",
			expectedPositions: []int{1},
			expectedDocuments: []string{"config:\n  data: |\n    ---\n    receivers: [otlp]\n"},
		},
		{
			name:              "Malformed_YieldedForTheCallerToReport",
			contents:          "kind: [unterminated\n",
			expectedPositions: []int{1},
			expectedDocuments: []string{"kind: [unterminated\n"},
		},
		{
			name:              "Empty_NoDocuments",
			contents:          "",
			expectedPositions: []int{},
			expectedDocuments: []string{},
		},
		{
			name:              "CommentsOnly_NoDocuments",
			contents:          "# nothing but a note\n",
			expectedPositions: []int{},
			expectedDocuments: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			positions := []int{}
			documents := []string{}
			for position, document := range YAMLStream(tt.contents).Documents() {
				positions = append(positions, position)
				documents = append(documents, string(document))
			}

			assert.Equal(t, tt.expectedPositions, positions)
			assert.Equal(t, tt.expectedDocuments, documents)
		})
	}
}

func TestNewYAMLStream(t *testing.T) {
	tests := []struct {
		name              string
		documents         []string
		expectedContents  string
		expectedDocuments int
	}{
		{
			name:              "None_Empty",
			documents:         []string{},
			expectedContents:  "",
			expectedDocuments: 0,
		},
		{
			name:              "SingleDocument_NoSeparator",
			documents:         []string{"kind: Installation\n"},
			expectedContents:  "kind: Installation\n",
			expectedDocuments: 1,
		},
		{
			name:              "TwoDocuments_Separated",
			documents:         []string{"kind: Installation\n", "kind: CollectionAgent\n"},
			expectedContents:  "kind: Installation\n---\nkind: CollectionAgent\n",
			expectedDocuments: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			documents := make([][]byte, 0, len(tt.documents))
			for _, document := range tt.documents {
				documents = append(documents, []byte(document))
			}

			stream := NewYAMLStream(documents)
			assert.Equal(t, tt.expectedContents, string(stream))

			// What the stream is built from, it reads back.
			read := 0
			for range stream.Documents() {
				read++
			}
			assert.Equal(t, tt.expectedDocuments, read)
		})
	}
}
