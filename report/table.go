package report

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/ftl/go-depend/model"
)

// table writes the metrics as an aligned table. Packages that violate an
// invariant are listed again below the table, with the details of their
// violations.
func table(w io.Writer, all []model.Metrics) error {
	if err := writeTable(w, all); err != nil {
		return err
	}
	return writeViolations(w, all)
}

func writeTable(w io.Writer, all []model.Metrics) error {
	writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	shown := shownColumns(all)

	fmt.Fprintln(writer, header(shown))
	for _, m := range all {
		fmt.Fprintln(writer, row(m, shown))
	}

	return writer.Flush()
}

func header(shown columns) string {
	result := "PACKAGE\tCA\tCE\tSTD\tEXT\tI\tA\tD\tZONE\tSDP"
	if shown.module {
		result = "MODULE\t" + result
	}
	if shown.history {
		result += "\tPERCEIVED\tDIFF"
	}
	return result
}

func row(m model.Metrics, shown columns) string {
	result := fmt.Sprintf("%s\t%d\t%d\t%d\t%d\t%.2f\t%.2f\t%.2f\t%s\t%s",
		m.Package.RelativePath(), m.Afferent, m.Efferent, m.Stdlib, m.External,
		m.Instability, m.Abstractness, m.Distance, zoneMarker(m.Zone), sdpMarker(m))
	if shown.module {
		result = m.Package.ModulePath + "\t" + result
	}
	if shown.history {
		result += fmt.Sprintf("\t%.2f\t%+.2f", *m.PerceivedInstability, m.InstabilityDifference())
	}
	return result
}

func writeViolations(w io.Writer, all []model.Metrics) error {
	var lines []string
	for _, m := range all {
		if m.Zone != model.MainSequence {
			lines = append(lines, fmt.Sprintf("  %s: %s, D=%.2f", m.Package.ImportPath, zoneName(m.Zone), m.Distance))
		}
		for _, dependency := range m.UnstableDependencies {
			lines = append(lines, fmt.Sprintf("  %s: imports the less stable package %s", m.Package.ImportPath, dependency))
		}
	}
	if len(lines) == 0 {
		return nil
	}

	_, err := fmt.Fprintf(w, "\nViolations:\n%s\n", strings.Join(lines, "\n"))
	return err
}

func countModules(all []model.Metrics) int {
	var modules []string
	for _, m := range all {
		if !slices.Contains(modules, m.Package.ModulePath) {
			modules = append(modules, m.Package.ModulePath)
		}
	}
	return len(modules)
}

func zoneMarker(zone model.Zone) string {
	switch zone {
	case model.ZoneOfPain:
		return "PAIN"
	case model.ZoneOfUselessness:
		return "USELESS"
	default:
		return "-"
	}
}

func zoneName(zone model.Zone) string {
	switch zone {
	case model.ZoneOfPain:
		return "zone of pain"
	case model.ZoneOfUselessness:
		return "zone of uselessness"
	default:
		return "main sequence"
	}
}

// sdpMarker returns the number of imports that violate the invariant about
// stable dependencies.
func sdpMarker(m model.Metrics) string {
	if len(m.UnstableDependencies) == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", len(m.UnstableDependencies))
}
