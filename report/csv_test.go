package report_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ftl/go-depend/model"
	"github.com/ftl/go-depend/report"
)

func TestCSV(t *testing.T) {
	core := metricsOf(module+"/core", 2, 1, 3, 1, 1.0/3.0, 0.25, model.MainSequence)
	core.Package.Dir = "core"
	core.Package.Types, core.Package.Funcs, core.Package.Abstract = 3, 1, 1
	core.UnstableDependencies = []string{module + "/helper", module + "/other"}

	assert.Equal(t, `module,import_path,dir,types,funcs,abstract,afferent,efferent,stdlib,external,instability,abstractness,signed_distance,distance,zone,unstable_dependencies
example.com/module,example.com/module/core,core,3,1,1,2,1,3,1,0.3333,0.2500,-0.4167,0.4167,main-sequence,example.com/module/helper;example.com/module/other
`, reportOf(t, report.CSVFormat, all(core)))
}

func TestCSVWithoutPackages(t *testing.T) {
	output := reportOf(t, report.CSVFormat, nil)

	assert.Equal(t, "module,import_path,dir,types,funcs,abstract,afferent,efferent,stdlib,external,instability,abstractness,signed_distance,distance,zone,unstable_dependencies\n", output)
}

func TestCSVZones(t *testing.T) {
	pain := metricsOf(module+"/pain", 2, 0, 0, 0, 0.0, 0.0, model.ZoneOfPain)
	useless := metricsOf(module+"/useless", 0, 0, 0, 0, 1.0, 1.0, model.ZoneOfUselessness)

	output := reportOf(t, report.CSVFormat, all(pain, useless))

	assert.Contains(t, output, ",pain,")
	assert.Contains(t, output, ",uselessness,")
}

func TestCSVWithTheRealityCheck(t *testing.T) {
	m := metricsOf(module+"/core", 2, 1, 0, 0, 0.25, 0.0, model.MainSequence)
	m.Package.Dir = "core"
	m.PerceivedInstability = value(0.5)

	output := reportOf(t, report.CSVFormat, all(m))

	assert.Contains(t, output, ",perceived_instability,instability_difference\n")
	assert.Contains(t, output, ",0.5000,0.2500\n")
}

func TestCSVWithoutTheRealityCheck(t *testing.T) {
	output := reportOf(t, report.CSVFormat, all(metricsOf(module+"/core", 2, 1, 0, 0, 0.25, 0.0, model.MainSequence)))

	assert.NotContains(t, output, "perceived_instability")
}

func TestUnknownFormat(t *testing.T) {
	err := report.Write(nil, report.Format("xml"), nil)

	assert.ErrorContains(t, err, `unknown format "xml"`)
	assert.ErrorContains(t, err, "table, json, csv")
}
