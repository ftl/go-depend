package report_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ftl/go-depend/model"
	"github.com/ftl/go-depend/report"
)

func TestJSON(t *testing.T) {
	core := metricsOf(module+"/core", 2, 1, 3, 1, 0.5, 0.25, model.ZoneOfPain)
	core.Package.Dir = "core"
	core.Package.Types, core.Package.Funcs, core.Package.Abstract = 3, 1, 1
	core.UnstableDependencies = []string{module + "/helper"}

	assert.JSONEq(t, `{
	  "packages": [
	    {
	      "module": "example.com/module",
	      "importPath": "example.com/module/core",
	      "dir": "core",
	      "types": 3,
	      "funcs": 1,
	      "abstract": 1,
	      "afferent": 2,
	      "efferent": 1,
	      "stdlib": 3,
	      "external": 1,
	      "instability": 0.5,
	      "abstractness": 0.25,
	      "abstractCoupling": 0,
	      "signedDistance": -0.25,
	      "distance": 0.25,
	      "zone": "pain",
	      "unstableDependencies": ["example.com/module/helper"]
	    }
	  ]
	}`, reportOf(t, report.JSONFormat, all(core)))
}

func TestJSONWithoutUnstableDependencies(t *testing.T) {
	output := reportOf(t, report.JSONFormat, all(metricsOf(module+"/leaf", 1, 0, 0, 0, 0.0, 1.0, model.MainSequence)))

	var parsed struct {
		Packages []struct {
			Zone                 string    `json:"zone"`
			UnstableDependencies *[]string `json:"unstableDependencies"`
		} `json:"packages"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &parsed))

	require.Len(t, parsed.Packages, 1)
	assert.Equal(t, "main-sequence", parsed.Packages[0].Zone)
	require.NotNil(t, parsed.Packages[0].UnstableDependencies, "an empty list, never null")
	assert.Empty(t, *parsed.Packages[0].UnstableDependencies)
}

func TestJSONWithTheRealityCheck(t *testing.T) {
	m := metricsOf(module+"/core", 2, 0, 0, 0, 0.0, 0.0, model.ZoneOfPain)
	m.PerceivedInstability = value(0.6)

	output := reportOf(t, report.JSONFormat, all(m))

	assert.Contains(t, output, `"perceivedInstability": 0.6`)
}

func TestJSONWithoutTheRealityCheck(t *testing.T) {
	output := reportOf(t, report.JSONFormat, all(metricsOf(module+"/core", 2, 0, 0, 0, 0.0, 0.0, model.MainSequence)))

	assert.NotContains(t, output, "perceivedInstability")
}

func TestJSONWithoutPackages(t *testing.T) {
	assert.JSONEq(t, `{"packages": []}`, reportOf(t, report.JSONFormat, nil))
}

func TestJSONZoneOfUselessness(t *testing.T) {
	output := reportOf(t, report.JSONFormat, all(metricsOf(module, 0, 0, 0, 0, 1.0, 1.0, model.ZoneOfUselessness)))

	assert.Contains(t, output, `"zone": "uselessness"`)
}

func reportOf(t *testing.T, format report.Format, all []model.Metrics) string {
	t.Helper()
	out := &bytes.Buffer{}
	require.NoError(t, report.Write(out, format, all))
	return out.String()
}
