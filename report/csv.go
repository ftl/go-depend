package report

import (
	"encoding/csv"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/ftl/go-depend/model"
)

// csvSeparator separates the unstable dependencies inside their single field.
const csvSeparator = ";"

var csvHeader = []string{
	"module", "import_path", "dir",
	"types", "funcs", "abstract",
	"afferent", "efferent", "stdlib", "external",
	"instability", "abstractness", "signed_distance", "distance",
	"zone", "unstable_dependencies",
}

// csvHistoryHeader contains the column that only exists if the git history
// was read.
var csvHistoryHeader = []string{"perceived_instability"}

func writeCSV(w io.Writer, all []model.Metrics) error {
	shown := shownColumns(all)

	writer := csv.NewWriter(w)
	if err := writer.Write(csvHeaderOf(shown)); err != nil {
		return err
	}
	for _, m := range all {
		if err := writer.Write(csvRecordOf(m, shown)); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}

func csvHeaderOf(shown columns) []string {
	if shown.history {
		return append(slices.Clone(csvHeader), csvHistoryHeader...)
	}
	return csvHeader
}

func csvRecordOf(m model.Metrics, shown columns) []string {
	result := csvMetricsOf(m)
	if shown.history {
		result = append(result, formatFloat(*m.PerceivedInstability))
	}
	return result
}

func csvMetricsOf(m model.Metrics) []string {
	return []string{
		m.Package.ModulePath,
		m.Package.ImportPath,
		m.Package.Dir,
		strconv.Itoa(m.Package.Types),
		strconv.Itoa(m.Package.Funcs),
		strconv.Itoa(m.Package.Abstract),
		strconv.Itoa(m.Afferent),
		strconv.Itoa(m.Efferent),
		strconv.Itoa(m.Stdlib),
		strconv.Itoa(m.External),
		formatFloat(m.Instability),
		formatFloat(m.Abstractness),
		formatFloat(m.SignedDistance),
		formatFloat(m.Distance),
		zoneToken(m.Zone),
		strings.Join(m.UnstableDependencies, csvSeparator),
	}
}

// formatFloat keeps more digits than the text table: a spreadsheet or another
// program can round the value itself.
func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 4, 64)
}
