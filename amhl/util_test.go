package amhl

import (
	"image/color"
	"testing"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func TestRndPayments(t *testing.T) {

	RndPayments()

}

func Test_find(t *testing.T) {
	pts_amhl := find("res_amhl_30k5.log")
	pts_union := find("res_union_30k5.log")

	p := plot.New()

	g_amhl, _ := plotter.NewScatter(pts_amhl)
	g_amhl.Color = color.RGBA{60, 179, 113, 255} // green
	g_union, _ := plotter.NewScatter(pts_union)
	g_union.Color = color.RGBA{106, 90, 205, 255} // purple

	p.Add(g_amhl, g_union)
	for i := 1000; i < 20000; i += 1000 {
		l, _ := plotter.NewLine(plotter.XYs{
			{X: float64(i), Y: 3}, {X: float64(i), Y: 10}})
		p.Add(l)
	}
	if err := p.Save(150*vg.Inch, 30*vg.Inch, "poly.png"); err != nil {
		panic(err)
	}
}

func Test_delegate(t *testing.T) {
	pts_amhl := delegate("res_amhl_30k5.log")
	println("-----")
	pts_union := delegate("res_union_30k5.log")

	p := plot.New()

	g_amhl, _ := plotter.NewScatter(pts_amhl)
	g_amhl.Color = color.RGBA{60, 179, 113, 255} // green
	g_union, _ := plotter.NewScatter(pts_union)
	g_union.Color = color.RGBA{106, 90, 205, 255} // purple

	p.Add(g_amhl, g_union)
	for i := 1000; i < 20000; i += 1000 {
		l, _ := plotter.NewLine(plotter.XYs{
			{X: float64(i), Y: 3}, {X: float64(i), Y: 10}})
		p.Add(l)
	}
	if err := p.Save(150*vg.Inch, 30*vg.Inch, "poly.png"); err != nil {
		panic(err)
	}

}
