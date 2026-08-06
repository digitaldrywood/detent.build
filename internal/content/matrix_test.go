package content

import "testing"

// The site claims the comparison matrix is copied from docs/comparison.md,
// including the rows Detent loses. A drifting copy makes that claim false, and
// a dropped row is how the claim broke the first time — the ~5-min setup row
// went missing. These assertions pin shape and the rows most likely to be
// quietly softened in Detent's favor.
//
// The source table lives in the Detent repository, which is not a dependency
// of this module, so this cannot diff against the file directly. It instead
// fails loudly if the transcription changes without someone re-reading the
// source.
func TestMatrixShapeMatchesSource(t *testing.T) {
	const wantRows = 14

	if len(Matrix) != wantRows {
		t.Errorf("matrix has %d rows, want %d — docs/comparison.md has %d capability rows and the site says it copies all of them",
			len(Matrix), wantRows, wantRows)
	}

	for _, row := range Matrix {
		if len(row.Cells) != len(Competitors) {
			t.Errorf("row %q has %d cells, want %d (one per competitor)",
				row.Capability, len(row.Cells), len(Competitors))
		}
	}
}

func TestMatrixMarksAreValid(t *testing.T) {
	valid := map[string]bool{"yes": true, "partial": true, "no": true, "na": true}

	for _, row := range Matrix {
		for i, cell := range row.Cells {
			if !valid[cell.Mark] {
				t.Errorf("row %q cell %d has invalid mark %q", row.Capability, i, cell.Mark)
			}
		}
	}
}

// Detent does not win every row, and the site says so out loud. If someone
// upgrades these cells, the claim on /why-detent becomes false.
func TestMatrixKeepsTheRowsDetentDoesNotWin(t *testing.T) {
	const detent = 0

	notWon := map[string]string{
		"Model-agnostic, BYO incl. local":          "partial",
		"Multi-channel triggers":                   "partial",
		"Local skills / workflows (your e2e etc.)": "yes",
	}

	for capability, wantMark := range notWon {
		row, ok := findRow(capability)
		if !ok {
			t.Errorf("row %q is missing from the matrix", capability)
			continue
		}
		if got := row.Cells[detent].Mark; got != wantMark {
			t.Errorf("row %q: Detent is marked %q, source says %q", capability, got, wantMark)
		}
	}

	// Competitors beat Detent outright on these; the cells must stay honest.
	beaten := []struct {
		capability string
		competitor int
		mark       string
	}{
		{"~5-min setup", 2, "yes"},           // Copilot agent: zero-install
		{"Budget / cost caps", 6, "yes"},     // Hyperagent: hosted controls
		{"Multi-channel triggers", 4, "yes"}, // Hermes: messaging gateway
	}

	for _, b := range beaten {
		row, ok := findRow(b.capability)
		if !ok {
			t.Errorf("row %q is missing from the matrix", b.capability)
			continue
		}
		if got := row.Cells[b.competitor].Mark; got != b.mark {
			t.Errorf("row %q: %s is marked %q, source says %q",
				b.capability, Competitors[b.competitor].Name, got, b.mark)
		}
	}
}

func TestMatrixAsOfMatchesSource(t *testing.T) {
	const want = "July 6, 2026"

	if MatrixAsOf != want {
		t.Errorf("MatrixAsOf = %q, want %q — update it only when the source's own date changes", MatrixAsOf, want)
	}
}

func findRow(capability string) (Row, bool) {
	for _, row := range Matrix {
		if row.Capability == capability {
			return row, true
		}
	}
	return Row{}, false
}
