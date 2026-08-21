package embedded

// Manifest mirrors the core_manifest.json written by cmd/dbasebuild.
type Manifest struct {
	Format            int             `json:"format"`
	GeneratorVersion  string          `json:"generator_version"`
	GeneratedAt       string          `json:"generated_at,omitempty"`
	FrequencySource   FrequencySource `json:"frequency_source"`
	TotalEntries      int             `json:"total_entries"`
	TotalUncompressed int64           `json:"bytes_uncompressed"`
	TotalCompressed   int64           `json:"bytes_compressed"`
	Files             []FileRecord    `json:"files"`
}

// FrequencySource records the ranking-data provenance.
type FrequencySource struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	License string `json:"license"`
	File    string `json:"file"`
	SHA256  string `json:"sha256"`
}

// FileRecord describes one embedded letter file.
type FileRecord struct {
	Name         string `json:"name"`
	SHA256       string `json:"sha256"`
	Entries      int    `json:"entries"`
	Uncompressed int64  `json:"bytes_uncompressed"`
	Compressed   int64  `json:"bytes_compressed"`
}
