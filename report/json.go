package report

import (
	"encoding/json"
	"io"

	"github.com/ftl/go-depend/model"
)

// jsonReport is the root of the JSON document. The packages are wrapped in an
// object so that further information can be added later without a change of
// the document structure.
type jsonReport struct {
	Packages []jsonPackage `json:"packages"`
}

// jsonPackage is the JSON representation of the metrics of one package. It is
// flat on purpose: a consumer needs no knowledge about the internal structure
// of go-depend.
type jsonPackage struct {
	Module     string `json:"module"`
	ImportPath string `json:"importPath"`
	Dir        string `json:"dir"`

	Types    int `json:"types"`
	Funcs    int `json:"funcs"`
	Abstract int `json:"abstract"`

	Afferent int `json:"afferent"`
	Efferent int `json:"efferent"`
	Stdlib   int `json:"stdlib"`
	External int `json:"external"`

	Instability    float64 `json:"instability"`
	Abstractness   float64 `json:"abstractness"`
	SignedDistance float64 `json:"signedDistance"`
	Distance       float64 `json:"distance"`

	Zone                 string   `json:"zone"`
	UnstableDependencies []string `json:"unstableDependencies"`

	// PerceivedInstability is only part of the document if the git history
	// was read.
	PerceivedInstability *float64 `json:"perceivedInstability,omitempty"`
}

func writeJSON(w io.Writer, all []model.Metrics) error {
	result := jsonReport{Packages: make([]jsonPackage, 0, len(all))}
	for _, m := range all {
		result.Packages = append(result.Packages, jsonPackageOf(m))
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func jsonPackageOf(m model.Metrics) jsonPackage {
	dependencies := m.UnstableDependencies
	if dependencies == nil {
		dependencies = []string{}
	}

	return jsonPackage{
		Module:               m.Package.ModulePath,
		ImportPath:           m.Package.ImportPath,
		Dir:                  m.Package.Dir,
		Types:                m.Package.Types,
		Funcs:                m.Package.Funcs,
		Abstract:             m.Package.Abstract,
		Afferent:             m.Afferent,
		Efferent:             m.Efferent,
		Stdlib:               m.Stdlib,
		External:             m.External,
		Instability:          m.Instability,
		Abstractness:         m.Abstractness,
		SignedDistance:       m.SignedDistance,
		Distance:             m.Distance,
		Zone:                 zoneToken(m.Zone),
		UnstableDependencies: dependencies,

		PerceivedInstability: m.PerceivedInstability,
	}
}
