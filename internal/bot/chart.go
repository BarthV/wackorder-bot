package bot

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"time"

	"github.com/barthv/wackorder-bot/internal/model"
)

// Chart layout constants.
const (
	chartW    = 800
	chartH    = 400
	chartPadL = 80
	chartPadR = 60
	chartPadT = 65
	chartPadB = 75
	chartDays = 30
	barGap    = 2
	fontScale = 3             // render each font pixel as a 3x3 block
	fontCellW = 6 * fontScale // scaled glyph width
	fontCellH = 6 * fontScale // scaled glyph height
	tickLen   = 6
)

var (
	colBg       = color.RGBA{0x2C, 0x2F, 0x33, 0xFF} // Discord dark
	colGrid     = color.RGBA{0x40, 0x44, 0x4B, 0xFF}
	colAxis     = color.RGBA{0x99, 0xAA, 0xB5, 0xFF}
	colBarOpen  = color.RGBA{0x58, 0x65, 0xF2, 0xFF} // blurple
	colBarReady = color.RGBA{0x57, 0xF2, 0x87, 0xFF} // green
	colLine     = color.RGBA{0xFE, 0xE7, 0x5C, 0xFF} // gold
	colLineDot  = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	colText     = color.RGBA{0xDC, 0xDD, 0xDE, 0xFF}
)

// generateRecapChart creates a PNG with:
//   - stacked bar histogram of pending orders by age (ordered vs ready)
//   - line overlay of cumulative done orders per day (last 30 days)
func generateRecapChart(pending, done []model.Order) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, chartW, chartH))
	draw.Draw(img, img.Bounds(), &image.Uniform{colBg}, image.Point{}, draw.Src)

	now := time.Now().UTC().Truncate(24 * time.Hour)
	plotW := chartW - chartPadL - chartPadR
	plotH := chartH - chartPadT - chartPadB

	// --- Pending histogram buckets (by creation age) ---
	type bucket struct{ ordered, ready int }
	buckets := make([]bucket, chartDays)
	for _, o := range pending {
		day := o.CreatedAt.UTC().Truncate(24 * time.Hour)
		age := int(now.Sub(day).Hours() / 24)
		idx := chartDays - 1 - age
		if idx < 0 {
			idx = 0
		}
		if idx >= chartDays {
			idx = chartDays - 1
		}
		switch o.Status {
		case model.StatusReady:
			buckets[idx].ready++
		default:
			buckets[idx].ordered++
		}
	}

	maxBar := 0
	for _, b := range buckets {
		if t := b.ordered + b.ready; t > maxBar {
			maxBar = t
		}
	}
	if maxBar == 0 {
		maxBar = 1
	}

	// --- Done line: count of orders completed per day (by updated_at) ---
	doneCounts := make([]int, chartDays)
	for _, o := range done {
		day := o.UpdatedAt.UTC().Truncate(24 * time.Hour)
		age := int(now.Sub(day).Hours() / 24)
		idx := chartDays - 1 - age
		if idx >= 0 && idx < chartDays {
			doneCounts[idx]++
		}
	}
	maxDone := 0
	for _, v := range doneCounts {
		if v > maxDone {
			maxDone = v
		}
	}
	if maxDone == 0 {
		maxDone = 1
	}

	// --- Draw grid lines ---
	for i := 0; i <= 4; i++ {
		y := chartPadT + plotH - (plotH*i)/4
		drawHLine(img, chartPadL, chartPadL+plotW, y, colGrid)
	}

	// --- Draw bars ---
	barW := plotW / chartDays
	if barW < 3 {
		barW = 3
	}
	for i, b := range buckets {
		x0 := chartPadL + i*barW + barGap
		x1 := chartPadL + (i+1)*barW - barGap
		if x1 <= x0 {
			x1 = x0 + 1
		}

		// Ordered (bottom)
		h1 := (b.ordered * plotH) / maxBar
		y1top := chartPadT + plotH - h1
		fillRect(img, x0, y1top, x1, chartPadT+plotH, colBarOpen)

		// Ready (stacked on top)
		h2 := (b.ready * plotH) / maxBar
		y2top := y1top - h2
		fillRect(img, x0, y2top, x1, y1top, colBarReady)
	}

	// --- Draw done line (right Y axis scale, step/staircase style) ---
	for i := 0; i < chartDays-1; i++ {
		x0 := chartPadL + i*barW
		x1 := chartPadL + (i+1)*barW
		y0 := chartPadT + plotH - (doneCounts[i]*plotH)/maxDone
		y1 := chartPadT + plotH - (doneCounts[i+1]*plotH)/maxDone
		drawThickLine(img, x0, y0, x1, y0, colLine) // horizontal step
		drawThickLine(img, x1, y0, x1, y1, colLine) // vertical step
	}
	// Dots on line (at start of each interval)
	for i, v := range doneCounts {
		if v > 0 {
			cx := chartPadL + i*barW
			cy := chartPadT + plotH - (v*plotH)/maxDone
			fillCircle(img, cx, cy, 5, colLineDot)
		}
	}

	// --- Axes ---
	drawHLine(img, chartPadL, chartPadL+plotW, chartPadT+plotH, colAxis)
	drawVLine(img, chartPadL, chartPadT, chartPadT+plotH, colAxis)

	// --- X-axis labels (every 5 days) ---
	for d := 0; d < chartDays; d += 5 {
		idx := chartDays - 1 - d
		x := chartPadL + idx*barW + barW/2
		drawVLine(img, x, chartPadT+plotH, chartPadT+plotH+tickLen, colAxis)
		label := "now"
		if d > 0 {
			label = fmt.Sprintf("-%dj", d)
		}
		drawString(img, x-len(label)*fontCellW/2, chartPadT+plotH+tickLen+4, label, colText)
	}

	// --- Left Y-axis labels (bar scale, deduplicated) ---
	seenL := make(map[int]bool)
	for i := 0; i <= 4; i++ {
		val := (maxBar * i) / 4
		if seenL[val] {
			continue
		}
		seenL[val] = true
		y := chartPadT + plotH - (plotH*i)/4
		label := fmt.Sprintf("%d", val)
		drawString(img, chartPadL-len(label)*fontCellW-6, y-fontCellH/2, label, colText)
	}

	// --- Right Y-axis labels (done scale, deduplicated) ---
	drawVLine(img, chartPadL+plotW, chartPadT, chartPadT+plotH, colAxis)
	seenR := make(map[int]bool)
	for i := 0; i <= 4; i++ {
		val := (maxDone * i) / 4
		if seenR[val] {
			continue
		}
		seenR[val] = true
		y := chartPadT + plotH - (plotH*i)/4
		label := fmt.Sprintf("%d", val)
		drawString(img, chartPadL+plotW+6, y-fontCellH/2, label, colLine)
	}

	// --- Title (slightly more spacing from plot) ---
	drawString(img, chartPadL, chartPadT-fontCellH-16, "Commandes", colText)

	// --- Legend (slightly more spacing from plot) ---
	legendY := chartH - fontCellH - 14
	sq := fontCellH // legend color square size
	gap := fontCellW / 2
	// Left-aligned: Ordered + Ready
	x := chartPadL
	fillRect(img, x, legendY, x+sq, legendY+sq, colBarOpen)
	x += sq + gap
	drawString(img, x, legendY, "Ordered", colText)
	x += 7*fontCellW + gap*2
	fillRect(img, x, legendY, x+sq, legendY+sq, colBarReady)
	x += sq + gap
	drawString(img, x, legendY, "Ready", colText)
	// Right-aligned: Done/jour
	doneLabel := "Completed/day"
	rx := chartPadL + plotW - len(doneLabel)*fontCellW
	fillRect(img, rx-sq-gap, legendY, rx-gap, legendY+sq, colLine)
	drawString(img, rx, legendY, doneLabel, colLine)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	return buf.Bytes(), nil
}

// --- drawing primitives ---

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if image.Pt(x, y).In(img.Bounds()) {
				img.Set(x, y, c)
			}
		}
	}
}

func drawHLine(img *image.RGBA, x0, x1, y int, c color.Color) {
	for x := x0; x <= x1; x++ {
		if image.Pt(x, y).In(img.Bounds()) {
			img.Set(x, y, c)
		}
	}
}

func drawVLine(img *image.RGBA, x, y0, y1 int, c color.Color) {
	for y := y0; y <= y1; y++ {
		if image.Pt(x, y).In(img.Bounds()) {
			img.Set(x, y, c)
		}
	}
}

// drawThickLine draws a line with uniform thickness in both directions.
func drawThickLine(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	const t = 2 // half-thickness => 5px total
	dx := x1 - x0
	dy := y1 - y0
	steps := abs(dx)
	if abs(dy) > steps {
		steps = abs(dy)
	}
	if steps == 0 {
		fillCircle(img, x0, y0, t, c)
		return
	}
	for i := 0; i <= steps; i++ {
		x := x0 + dx*i/steps
		y := y0 + dy*i/steps
		for tx := -t; tx <= t; tx++ {
			for ty := -t; ty <= t; ty++ {
				if image.Pt(x+tx, y+ty).In(img.Bounds()) {
					img.Set(x+tx, y+ty, c)
				}
			}
		}
	}
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	dx := x1 - x0
	dy := y1 - y0
	steps := abs(dx)
	if abs(dy) > steps {
		steps = abs(dy)
	}
	if steps == 0 {
		img.Set(x0, y0, c)
		return
	}
	const thickness = 2 // half-thickness: draws -t to +t => 2*t+1 = 5 pixels wide
	for i := 0; i <= steps; i++ {
		x := x0 + dx*i/steps
		y := y0 + dy*i/steps
		for t := -thickness; t <= thickness; t++ {
			if image.Pt(x, y+t).In(img.Bounds()) {
				img.Set(x, y+t, c)
			}
		}
	}
}

func fillCircle(img *image.RGBA, cx, cy, r int, c color.Color) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				x, y := cx+dx, cy+dy
				if image.Pt(x, y).In(img.Bounds()) {
					img.Set(x, y, c)
				}
			}
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// drawString renders text using a minimal built-in 5x7 bitmap font, scaled by fontScale.
func drawString(img *image.RGBA, x, y int, s string, c color.Color) {
	for _, ch := range s {
		glyph := getGlyph(ch)
		for row := 0; row < 7; row++ {
			for col := 0; col < 5; col++ {
				if glyph[row]&(1<<(4-col)) != 0 {
					// Draw a fontScale x fontScale block for each pixel
					for dy := 0; dy < fontScale; dy++ {
						for dx := 0; dx < fontScale; dx++ {
							px, py := x+col*fontScale+dx, y+row*fontScale+dy
							if image.Pt(px, py).In(img.Bounds()) {
								img.Set(px, py, c)
							}
						}
					}
				}
			}
		}
		x += fontCellW
	}
}

// getGlyph returns a 7-row bitmap (5 bits per row) for basic ASCII chars.
func getGlyph(ch rune) [7]byte {
	if int(ch) < len(font5x7) && font5x7[ch] != [7]byte{} {
		return font5x7[ch]
	}
	return font5x7['?']
}

// font5x7 is a minimal 5x7 bitmap font covering digits, uppercase, lowercase, and punctuation.
var font5x7 [128][7]byte

func init() {
	// Digits
	font5x7['0'] = [7]byte{0x0E, 0x11, 0x13, 0x15, 0x19, 0x11, 0x0E}
	font5x7['1'] = [7]byte{0x04, 0x0C, 0x04, 0x04, 0x04, 0x04, 0x0E}
	font5x7['2'] = [7]byte{0x0E, 0x11, 0x01, 0x06, 0x08, 0x10, 0x1F}
	font5x7['3'] = [7]byte{0x0E, 0x11, 0x01, 0x06, 0x01, 0x11, 0x0E}
	font5x7['4'] = [7]byte{0x02, 0x06, 0x0A, 0x12, 0x1F, 0x02, 0x02}
	font5x7['5'] = [7]byte{0x1F, 0x10, 0x1E, 0x01, 0x01, 0x11, 0x0E}
	font5x7['6'] = [7]byte{0x06, 0x08, 0x10, 0x1E, 0x11, 0x11, 0x0E}
	font5x7['7'] = [7]byte{0x1F, 0x01, 0x02, 0x04, 0x08, 0x08, 0x08}
	font5x7['8'] = [7]byte{0x0E, 0x11, 0x11, 0x0E, 0x11, 0x11, 0x0E}
	font5x7['9'] = [7]byte{0x0E, 0x11, 0x11, 0x0F, 0x01, 0x02, 0x0C}
	// Uppercase
	font5x7['A'] = [7]byte{0x0E, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11}
	font5x7['B'] = [7]byte{0x1E, 0x11, 0x11, 0x1E, 0x11, 0x11, 0x1E}
	font5x7['C'] = [7]byte{0x0E, 0x11, 0x10, 0x10, 0x10, 0x11, 0x0E}
	font5x7['D'] = [7]byte{0x1E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x1E}
	font5x7['E'] = [7]byte{0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x1F}
	font5x7['F'] = [7]byte{0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x10}
	font5x7['G'] = [7]byte{0x0E, 0x11, 0x10, 0x17, 0x11, 0x11, 0x0E}
	font5x7['H'] = [7]byte{0x11, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11}
	font5x7['I'] = [7]byte{0x0E, 0x04, 0x04, 0x04, 0x04, 0x04, 0x0E}
	font5x7['J'] = [7]byte{0x07, 0x02, 0x02, 0x02, 0x02, 0x12, 0x0C}
	font5x7['K'] = [7]byte{0x11, 0x12, 0x14, 0x18, 0x14, 0x12, 0x11}
	font5x7['L'] = [7]byte{0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x1F}
	font5x7['M'] = [7]byte{0x11, 0x1B, 0x15, 0x15, 0x11, 0x11, 0x11}
	font5x7['N'] = [7]byte{0x11, 0x19, 0x15, 0x13, 0x11, 0x11, 0x11}
	font5x7['O'] = [7]byte{0x0E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E}
	font5x7['P'] = [7]byte{0x1E, 0x11, 0x11, 0x1E, 0x10, 0x10, 0x10}
	font5x7['Q'] = [7]byte{0x0E, 0x11, 0x11, 0x11, 0x15, 0x12, 0x0D}
	font5x7['R'] = [7]byte{0x1E, 0x11, 0x11, 0x1E, 0x14, 0x12, 0x11}
	font5x7['S'] = [7]byte{0x0E, 0x11, 0x10, 0x0E, 0x01, 0x11, 0x0E}
	font5x7['T'] = [7]byte{0x1F, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04}
	font5x7['U'] = [7]byte{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E}
	font5x7['V'] = [7]byte{0x11, 0x11, 0x11, 0x11, 0x0A, 0x0A, 0x04}
	font5x7['W'] = [7]byte{0x11, 0x11, 0x11, 0x15, 0x15, 0x1B, 0x11}
	font5x7['X'] = [7]byte{0x11, 0x11, 0x0A, 0x04, 0x0A, 0x11, 0x11}
	font5x7['Y'] = [7]byte{0x11, 0x11, 0x0A, 0x04, 0x04, 0x04, 0x04}
	font5x7['Z'] = [7]byte{0x1F, 0x01, 0x02, 0x04, 0x08, 0x10, 0x1F}
	// Lowercase
	font5x7['a'] = [7]byte{0x00, 0x00, 0x0E, 0x01, 0x0F, 0x11, 0x0F}
	font5x7['b'] = [7]byte{0x10, 0x10, 0x1E, 0x11, 0x11, 0x11, 0x1E}
	font5x7['c'] = [7]byte{0x00, 0x00, 0x0E, 0x11, 0x10, 0x11, 0x0E}
	font5x7['d'] = [7]byte{0x01, 0x01, 0x0F, 0x11, 0x11, 0x11, 0x0F}
	font5x7['e'] = [7]byte{0x00, 0x00, 0x0E, 0x11, 0x1F, 0x10, 0x0E}
	font5x7['f'] = [7]byte{0x06, 0x09, 0x08, 0x1C, 0x08, 0x08, 0x08}
	font5x7['g'] = [7]byte{0x00, 0x00, 0x0F, 0x11, 0x0F, 0x01, 0x0E}
	font5x7['h'] = [7]byte{0x10, 0x10, 0x1E, 0x11, 0x11, 0x11, 0x11}
	font5x7['i'] = [7]byte{0x04, 0x00, 0x0C, 0x04, 0x04, 0x04, 0x0E}
	font5x7['j'] = [7]byte{0x02, 0x00, 0x06, 0x02, 0x02, 0x12, 0x0C}
	font5x7['k'] = [7]byte{0x10, 0x10, 0x12, 0x14, 0x18, 0x14, 0x12}
	font5x7['l'] = [7]byte{0x0C, 0x04, 0x04, 0x04, 0x04, 0x04, 0x0E}
	font5x7['m'] = [7]byte{0x00, 0x00, 0x1A, 0x15, 0x15, 0x11, 0x11}
	font5x7['n'] = [7]byte{0x00, 0x00, 0x1E, 0x11, 0x11, 0x11, 0x11}
	font5x7['o'] = [7]byte{0x00, 0x00, 0x0E, 0x11, 0x11, 0x11, 0x0E}
	font5x7['p'] = [7]byte{0x00, 0x00, 0x1E, 0x11, 0x1E, 0x10, 0x10}
	font5x7['q'] = [7]byte{0x00, 0x00, 0x0F, 0x11, 0x0F, 0x01, 0x01}
	font5x7['r'] = [7]byte{0x00, 0x00, 0x16, 0x19, 0x10, 0x10, 0x10}
	font5x7['s'] = [7]byte{0x00, 0x00, 0x0F, 0x10, 0x0E, 0x01, 0x1E}
	font5x7['t'] = [7]byte{0x08, 0x08, 0x1C, 0x08, 0x08, 0x09, 0x06}
	font5x7['u'] = [7]byte{0x00, 0x00, 0x11, 0x11, 0x11, 0x11, 0x0F}
	font5x7['v'] = [7]byte{0x00, 0x00, 0x11, 0x11, 0x0A, 0x0A, 0x04}
	font5x7['w'] = [7]byte{0x00, 0x00, 0x11, 0x11, 0x15, 0x15, 0x0A}
	font5x7['x'] = [7]byte{0x00, 0x00, 0x11, 0x0A, 0x04, 0x0A, 0x11}
	font5x7['y'] = [7]byte{0x00, 0x00, 0x11, 0x11, 0x0F, 0x01, 0x0E}
	font5x7['z'] = [7]byte{0x00, 0x00, 0x1F, 0x02, 0x04, 0x08, 0x1F}
	// Punctuation & symbols
	font5x7[' '] = [7]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	font5x7['-'] = [7]byte{0x00, 0x00, 0x00, 0x1F, 0x00, 0x00, 0x00}
	font5x7['+'] = [7]byte{0x00, 0x04, 0x04, 0x1F, 0x04, 0x04, 0x00}
	font5x7['/'] = [7]byte{0x01, 0x01, 0x02, 0x04, 0x08, 0x10, 0x10}
	font5x7['('] = [7]byte{0x02, 0x04, 0x08, 0x08, 0x08, 0x04, 0x02}
	font5x7[')'] = [7]byte{0x08, 0x04, 0x02, 0x02, 0x02, 0x04, 0x08}
	font5x7['.'] = [7]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x0C}
	font5x7[','] = [7]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x08}
	font5x7['?'] = [7]byte{0x0E, 0x11, 0x01, 0x02, 0x04, 0x00, 0x04}
	font5x7['!'] = [7]byte{0x04, 0x04, 0x04, 0x04, 0x04, 0x00, 0x04}
	font5x7['#'] = [7]byte{0x0A, 0x0A, 0x1F, 0x0A, 0x1F, 0x0A, 0x0A}
	font5x7[':'] = [7]byte{0x00, 0x0C, 0x0C, 0x00, 0x0C, 0x0C, 0x00}

}
