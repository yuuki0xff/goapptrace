package render

import (
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	"math"
	"strings"

	"github.com/yuuki0xff/goapptrace/tracer/log"
)

type ColorRule int

const (
	ColoringPerGoroutine ColorRule = iota
	ColoringPerFunction
	ColoringPerModule
)

var (
	ColorRuleNames = map[string]ColorRule{
		"goroutine": ColoringPerGoroutine,
		"function":  ColoringPerFunction,
		"module":    ColoringPerModule,
	}
)

type Colors struct {
	ColorRule ColorRule
	NColors   int
	strColors []string
	hash      hash.Hash64
}

func (c *Colors) GetByGoroutine(gr *log.Goroutine) string {
	switch c.ColorRule {
	case ColoringPerGoroutine:
		return c.GetByInt(int(gr.GID))
	default:
		return c.GetByFuncLog(gr.Records[0])
	}
}

func (c *Colors) GetByFuncLog(fl *log.FuncLog) string {
	switch c.ColorRule {
	case ColoringPerGoroutine:
		return c.GetByInt(int(fl.GID))
	case ColoringPerFunction:
		return c.GetByString(fl.Frames[0].Function)
	case ColoringPerModule:
		f := fl.Frames[0].Function

		// strip function name from f
		moduleHierarchy := strings.Split(f, "/")
		last := len(moduleHierarchy) - 1
		moduleHierarchy[last] = strings.SplitN(moduleHierarchy[last], ".", 2)[0]
		moduleName := strings.Join(moduleHierarchy, "/")

		return c.GetByString(moduleName)
	default:
		panic("Unsupported rule")
	}
}

func (c *Colors) GetByInt(value int) string {
	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, 31415926)
	return c.GetByBytes(bs)
}

func (c *Colors) GetByString(name string) string {
	return c.GetByBytes([]byte(name))
}

func (c *Colors) GetByBytes(value []byte) string {
	if c.strColors == nil {
		c.Init()
	}
	if _, err := c.hash.Write(value); err != nil {
		panic(err)
	}
	return c.strColors[c.hash.Sum64()%uint64(c.NColors)]
}

func (c *Colors) Init() {
	c.strColors = generateColors(c.NColors, 0.6, 0.7)
	c.hash = fnv.New64()
}

func generateColors(ncolors int, s, v float64) []string {
	if ncolors < 0 {
		panic("ncolors must be less than 0.")
	}
	if s > v {
		panic("s <= v is not satisfied.")
	}

	colors := make([]string, ncolors)
	for i := range colors {
		h := float64(i) / float64(ncolors)
		colors[i] = colorStr(hsv2rgb(h, s, v))
	}
	return colors
}

// convert color space from HSV to RGB
// 0.0 <= h,s,v <= 1.0
// 0 <= return_value <= 255
func hsv2rgb(h, s, v float64) [3]int {
	hh := int(360 * h / 60)
	c := s
	x := c * (1 - math.Abs(float64((hh%2)-1)))
	convertTable := [][3]float64{
		{c, x, 0},
		{x, c, 0},
		{0, c, x},
		{0, x, c},
		{x, 0, c},
		{c, 0, x},
	}

	rgb := convertTable[hh]
	return [3]int{
		int(255 * (v - c + rgb[0])), // R
		int(255 * (v - c + rgb[1])), // G
		int(255 * (v - c + rgb[2])), // B
	}
}

func colorStr(rgb [3]int) string {
	return fmt.Sprintf("#%02x%02x%02x",
		rgb[0], rgb[1], rgb[2])
}
