package controller

import (
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// applyStandardExcelTableStyle applies a consistent table look for export sheets.
func applyStandardExcelTableStyle(f *excelize.File, sheet string, headerRow, lastRow, lastVisibleCol int, colWidth map[int]float64) {
	if headerRow <= 0 || lastVisibleCol <= 0 {
		return
	}
	lastVisibleColName, _ := excelize.CoordinatesToCellName(lastVisibleCol, headerRow)

	// Header style + frozen header + filter
	if st, e := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#1F2937"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#E8EEF7"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	}); e == nil {
		start, _ := excelize.CoordinatesToCellName(1, headerRow)
		_ = f.SetCellStyle(sheet, start, lastVisibleColName, st)
	}
	_ = f.SetRowHeight(sheet, headerRow, 24)
	topLeft, _ := excelize.CoordinatesToCellName(1, headerRow+1)
	_ = f.SetPanes(sheet, &excelize.Panes{Freeze: true, Split: false, XSplit: 0, YSplit: headerRow, TopLeftCell: topLeft, ActivePane: "bottomLeft"})
	_ = f.AutoFilter(sheet, "A"+strconv.Itoa(headerRow)+":"+lastVisibleColName, nil)

	if lastRow > headerRow {
		dataStart, _ := excelize.CoordinatesToCellName(1, headerRow+1)
		dataEnd, _ := excelize.CoordinatesToCellName(lastVisibleCol, lastRow)
		if st, e := f.NewStyle(&excelize.Style{
			Alignment: &excelize.Alignment{Vertical: "center"},
		}); e == nil {
			_ = f.SetCellStyle(sheet, dataStart, dataEnd, st)
		}
		if zebra, e := f.NewStyle(&excelize.Style{
			Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#F8FAFC"}},
		}); e == nil {
			for r := headerRow + 1; r <= lastRow; r++ {
				if r%2 == 0 {
					start, _ := excelize.CoordinatesToCellName(1, r)
					end, _ := excelize.CoordinatesToCellName(lastVisibleCol, r)
					_ = f.SetCellStyle(sheet, start, end, zebra)
				}
			}
		}
	}

	// Visible columns widths
	for col := 1; col <= lastVisibleCol; col++ {
		cellName, _ := excelize.CoordinatesToCellName(col, headerRow)
		colLetter := strings.TrimRight(cellName, "0123456789")
		w := colWidth[col]
		if w <= 0 {
			w = 12
		}
		_ = f.SetColWidth(sheet, colLetter, colLetter, w)
	}
}
