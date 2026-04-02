package main

import (
	"fmt"
	"os"
	"strings"

	"adt-testserver/endpoints"

	"gopkg.in/yaml.v3"
)

// fixtureFile mirrors the YAML fixture format.
type fixtureFile struct {
	// Sources maps ADT path to ABAP source content.
	// Example:
	//   sources:
	//     /sap/bc/adt/programs/programs/ZTEST/source/main: |
	//       REPORT ztest.
	//       WRITE: / 'Hello'.
	Sources map[string]string `yaml:"sources"`

	// Locks maps object URL to a pre-existing lock handle.
	// Example:
	//   locks:
	//     /sap/bc/adt/programs/programs/ZLOCKED: MY_LOCK_HANDLE
	Locks map[string]string `yaml:"locks"`

	// DataPreview maps an exact SQL query string to a slice of column→value rows.
	// Example:
	//   datapreview:
	//     "SELECT * FROM T000 WHERE MANDT = '001'":
	//       - MANDT: "001"
	//         MTEXT: "Test Client"
	//         LOGSYS: "T01CLNT001"
	DataPreview map[string][]map[string]string `yaml:"datapreview"`
}

// loadFixtures reads a YAML fixture file and populates the shared state.
func loadFixtures(path string, s *endpoints.State) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading fixture file: %w", err)
	}

	var f fixtureFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("parsing fixture file: %w", err)
	}

	for path, src := range f.Sources {
		if strings.Contains(path, "{") {
			if err := s.AddPatternSource(path, src); err != nil {
				return fmt.Errorf("compiling source pattern %q: %w", path, err)
			}
		} else {
			s.SetSource(path, src)
		}
	}
	for objectURL, handle := range f.Locks {
		s.AcquireLock(objectURL, handle)
	}
	for query, rows := range f.DataPreview {
		s.SetDataPreview(query, rows)
	}

	return nil
}
