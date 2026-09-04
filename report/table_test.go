package report_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ftl/go-depend/model"
	"github.com/ftl/go-depend/report"
)

const module = "example.com/module"

func TestTable(t *testing.T) {
	all := []model.Metrics{
		metricsOf(module, 0, 2, 3, 1, 1.0, 0.0, model.MainSequence),
		metricsOf(module+"/leaf", 4, 0, 1, 0, 0.0, 0.0, model.ZoneOfPain),
	}

	assert.Equal(t, `PACKAGE  CA  CE  STD  EXT  I     A     D     ZONE  SDP
.        0   2   3    1    1.00  0.00  0.00  -     -
leaf     4   0   1    0    0.00  0.00  1.00  PAIN  -

Violations:
  example.com/module/leaf: zone of pain, D=1.00
`, tableOf(t, all))
}

func TestTableWithoutViolations(t *testing.T) {
	all := []model.Metrics{metricsOf(module+"/hub", 1, 1, 0, 0, 0.5, 0.5, model.MainSequence)}

	assert.Equal(t, `PACKAGE  CA  CE  STD  EXT  I     A     D     ZONE  SDP
hub      1   1   0    0    0.50  0.50  0.00  -     -
`, tableOf(t, all))
}

func TestTableWithUnstableDependencies(t *testing.T) {
	core := metricsOf(module+"/core", 2, 2, 0, 0, 0.5, 0.0, model.MainSequence)
	core.UnstableDependencies = []string{module + "/helper", module + "/other"}
	useless := metricsOf(module+"/abstract", 0, 0, 0, 0, 1.0, 1.0, model.ZoneOfUselessness)

	assert.Equal(t, `PACKAGE   CA  CE  STD  EXT  I     A     D     ZONE     SDP
abstract  0   0   0    0    1.00  1.00  1.00  USELESS  -
core      2   2   0    0    0.50  0.00  0.50  -        2

Violations:
  example.com/module/abstract: zone of uselessness, D=1.00
  example.com/module/core: imports the less stable package example.com/module/helper
  example.com/module/core: imports the less stable package example.com/module/other
`, tableOf(t, all(useless, core)))
}

func TestTableGroupsByModule(t *testing.T) {
	const other = "example.com/other"
	first := metricsOf(module+"/pkg", 0, 0, 0, 0, 1.0, 0.0, model.MainSequence)
	second := model.Metrics{Package: model.Package{ImportPath: other + "/pkg", ModulePath: other}}

	output := tableOf(t, all(second, first))

	assert.Equal(t, `MODULE              PACKAGE  CA  CE  STD  EXT  I     A     D     ZONE  SDP
example.com/module  pkg      0   0   0    0    1.00  0.00  0.00  -     -
example.com/other   pkg      0   0   0    0    0.00  0.00  0.00  -     -
`, output)
}

func TestTableGroupsNestedModules(t *testing.T) {
	// The import path of a package always starts with the path of its module.
	// Only nested modules make the order by module differ from the order by
	// import path: zzz belongs to the outer module and comes first, although
	// its import path sorts after the import path of aaa.
	const (
		outer = "example.com/m"
		inner = "example.com/m/sub"
	)
	outerPkg := model.Metrics{Package: model.Package{ImportPath: outer + "/zzz", ModulePath: outer}}
	innerPkg := model.Metrics{Package: model.Package{ImportPath: inner + "/aaa", ModulePath: inner}}

	output := tableOf(t, all(innerPkg, outerPkg))

	assert.Equal(t, `MODULE             PACKAGE  CA  CE  STD  EXT  I     A     D     ZONE  SDP
example.com/m      zzz      0   0   0    0    0.00  0.00  0.00  -     -
example.com/m/sub  aaa      0   0   0    0    0.00  0.00  0.00  -     -
`, output)
}

func TestTableWithTheRealityCheck(t *testing.T) {
	stable := metricsOf(module+"/stable", 3, 0, 0, 0, 0.0, 0.5, model.MainSequence)
	stable.PerceivedInstability = value(0.75)
	churning := metricsOf(module+"/churn", 0, 2, 0, 0, 1.0, 0.0, model.MainSequence)
	churning.PerceivedInstability = value(0.25)

	assert.Equal(t, `PACKAGE  CA  CE  STD  EXT  I     A     D     ZONE  SDP  PERCEIVED  DIFF
churn    0   2   0    0    1.00  0.00  0.00  -     -    0.25       -0.75
stable   3   0   0    0    0.00  0.50  0.50  -     -    0.75       +0.75
`, tableOf(t, all(stable, churning)))
}

func TestTableWithoutTheRealityCheck(t *testing.T) {
	output := tableOf(t, all(metricsOf(module+"/pkg", 0, 0, 0, 0, 1.0, 0.0, model.MainSequence)))

	assert.NotContains(t, output, "PERCEIVED")
	assert.NotContains(t, output, "DIFF")
}

func value(v float64) *float64 {
	return &v
}

func TestTableWithoutPackages(t *testing.T) {
	assert.Equal(t, "PACKAGE  CA  CE  STD  EXT  I  A  D  ZONE  SDP\n", tableOf(t, nil))
}

func metricsOf(importPath string, afferent, efferent, stdlib, external int, instability, abstractness float64, zone model.Zone) model.Metrics {
	signed := abstractness + instability - 1
	distance := signed
	if distance < 0 {
		distance = -distance
	}
	return model.Metrics{
		Package:        model.Package{ImportPath: importPath, ModulePath: module},
		Afferent:       afferent,
		Efferent:       efferent,
		Stdlib:         stdlib,
		External:       external,
		Instability:    instability,
		Abstractness:   abstractness,
		SignedDistance: signed,
		Distance:       distance,
		Zone:           zone,
	}
}

func all(metrics ...model.Metrics) []model.Metrics {
	return metrics
}

func tableOf(t *testing.T, all []model.Metrics) string {
	t.Helper()
	out := &bytes.Buffer{}
	require.NoError(t, report.Write(out, report.TableFormat, all))
	return out.String()
}
