package domain

import (
	"bufio"
	"bytes"
	"iter"

	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	kyaml "sigs.k8s.io/yaml"
)

// documentSeparator stands between the documents of a YAML stream.
const documentSeparator = "---\n"

// YAMLStream is a YAML byte stream read as the documents it holds.
type YAMLStream []byte

// NewYAMLStream joins documents into one stream. Marshalled documents end in a
// newline, so the separator adds none.
func NewYAMLStream(documents [][]byte) YAMLStream {
	return bytes.Join(documents, []byte(documentSeparator))
}

// Documents yields every document that declares something, with the position a
// reader counts in the file. Blank and comment-only documents are skipped, and
// still counted. Documents are not parsed, so a caller can report which one is
// malformed.
func (s YAMLStream) Documents() iter.Seq2[int, []byte] {
	return func(yield func(int, []byte) bool) {
		reader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(s)))

		// Reading a byte slice fails only at the end of the stream.
		for position := 1; ; position++ {
			document, err := reader.Read()
			if err != nil {
				return
			}

			if isEmptyYAMLDocument(document) {
				continue
			}

			if !yield(position, document) {
				return
			}
		}
	}
}

// isEmptyYAMLDocument reports a document that declares nothing: blank or
// comments only. Malformed contents are not empty; the caller reports those
// against the document they came from.
func isEmptyYAMLDocument(document []byte) bool {
	var contents any
	if err := kyaml.Unmarshal(document, &contents); err != nil {
		return false
	}

	return contents == nil
}
