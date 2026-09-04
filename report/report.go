// Package report renders the metrics of the analyzed packages. The text table
// is meant for a reader, the JSON and CSV formats for another program.
package report

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/ftl/go-depend/model"
)

// Format is the output format of a report.
type Format string

const (
	// TableFormat is an aligned text table for a reader.
	TableFormat Format = "table"
	// JSONFormat is a JSON document for another program.
	JSONFormat Format = "json"
	// CSVFormat is a CSV document for a spreadsheet.
	CSVFormat Format = "csv"
)

// Formats returns all supported formats, in the order of their preference.
func Formats() []Format {
	return []Format{TableFormat, JSONFormat, CSVFormat}
}

// Write renders the metrics in the given format, grouped by module and sorted
// by the import path of the package.
func Write(w io.Writer, format Format, all []model.Metrics) error {
	sorted := sortedForReport(all)

	switch format {
	case TableFormat:
		return table(w, sorted)
	case JSONFormat:
		return writeJSON(w, sorted)
	case CSVFormat:
		return writeCSV(w, sorted)
	default:
		return fmt.Errorf("unknown format %q, use one of %s", format, strings.Join(FormatNames(), ", "))
	}
}

// FormatNames returns the names of all supported formats, for the help of a
// command line flag.
func FormatNames() []string {
	result := make([]string, 0, len(Formats()))
	for _, format := range Formats() {
		result = append(result, string(format))
	}
	return result
}

// columns are the optional columns of a report.
type columns struct {
	// module is true if the report covers more than one module, for example
	// in a workspace.
	module bool
	// history is true if the git history was read.
	history bool
}

func shownColumns(all []model.Metrics) columns {
	result := columns{module: countModules(all) > 1}
	for _, m := range all {
		if m.PerceivedInstability != nil {
			result.history = true
			break
		}
	}
	return result
}

// sortedForReport returns the metrics sorted by module, and within a module by
// the import path of the package. Only nested modules make this order differ
// from the order by import path.
func sortedForReport(all []model.Metrics) []model.Metrics {
	result := slices.Clone(all)
	slices.SortFunc(result, func(a, b model.Metrics) int {
		return cmp.Or(
			cmp.Compare(a.Package.ModulePath, b.Package.ModulePath),
			cmp.Compare(a.Package.ImportPath, b.Package.ImportPath),
		)
	})
	return result
}

// zoneToken is the name of the zone in the machine readable formats.
func zoneToken(zone model.Zone) string {
	switch zone {
	case model.ZoneOfPain:
		return "pain"
	case model.ZoneOfUselessness:
		return "uselessness"
	default:
		return "main-sequence"
	}
}
