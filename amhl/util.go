package amhl

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/montanaflynn/stats"
	"gonum.org/v1/plot/plotter"
)

func RndPayments() {
	f, err := os.Create("data/payments_30k.txt") // creating...
	if err != nil {
		fmt.Printf("error creating file: %v", err)
		return
	}
	defer f.Close()
	count := 0
	nodes := []int{6, 5, 7, 9, 11}
	for count < TX_SIMULATE {
		payer := rand.Intn(len(nodes))
		payee := rand.Intn(len(nodes))

		if payer != payee {
			_, err = f.WriteString(fmt.Sprintf("%d %d\n", nodes[payer], nodes[payee]))
			if err != nil {
				fmt.Printf("error writing string: %v", err)
			}
			count++
		}
	}
}

func find(file string) plotter.XYs {
	f, _ := os.Open(file)
	scanner := bufio.NewScanner(f)
	txts := make(map[string]float64)
	for i := 0; i < TX_SIMULATE; i++ {
		txts[fmt.Sprintf("tx%d", i+1)] = -1
	}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "tx success") {
			re := regexp.MustCompile("tx[0-9]+")
			re2 := regexp.MustCompile("[0-9]+.[0-9]+s")
			f := re.FindAllString(line, -1)
			f2 := re2.FindAllString(line, -1)
			val, _ := txts[f[0]]
			if val == -1 {
				txts[f[0]], _ = strconv.ParseFloat(strings.TrimSuffix(f2[0], "s"), 64)
			}

		}
	}

	println("----- ")
	data := []float64{}
	var pts plotter.XYs

	for i := 0; i < TX_SIMULATE; i++ {
		if txts[fmt.Sprintf("tx%d", i+1)] == -1 {
			print(fmt.Sprintf("tx%d", i+1), " ")
		} else {
			v := float64(txts[fmt.Sprintf("tx%d", i+1)])
			// vp := float64(txts[fmt.Sprintf("tx%d", i)])
			data = append(data, v)
			pts = append(pts, plotter.XY{X: float64(i), Y: v})
			// if i > 0 && v-vp > 3.0 && vp > 0 {
			// 	println(i, v, vp)
			// }

		}
	}
	// temp := []float64{}
	// for i := 0; i < len(txts); i++ {
	// 	temp = append(temp, txts[fmt.Sprintf("tx%d", i+1)])
	// 	if i%100 == 0 {
	// 		mean, _ := stats.Mean(temp)
	// 		println(i/100, mean)
	// 		temp = nil
	// 	}
	// }
	mean, _ := stats.Mean(data)
	std, _ := stats.StandardDeviation(data)

	println("mean", mean, "std", std)
	return pts
}

func delegate(file string) plotter.XYs {
	f, _ := os.Open(file)
	scanner := bufio.NewScanner(f)
	txts := make(map[float64]time.Time)
	data := make(map[int]float64)
	var pts plotter.XYs
	for i := 0; i < TX_SIMULATE; i++ {
		txts[float64(i+1)] = time.Now()
		data[i+1] = 0

	}
	reFrom := regexp.MustCompile(`\d{1,2} > \d{1,2}`)
	reTx := regexp.MustCompile("tx[0-9]+")
	reDate := regexp.MustCompile(`\d{2}:\d{2}:\d{2}.\d{6}`)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "TX_FROM_BACK") || strings.Contains(line, "C_SIG_FROM_FRONT") {
			from := reFrom.FindAllString(line, -1)
			dateS := reDate.FindAllString(line, -1)[0]

			date, _ := time.Parse("15:04:05.000000", dateS)

			txno := reTx.FindAllString(line, -1)[0]
			t, _ := strings.CutPrefix(txno, "tx")
			no, _ := strconv.Atoi(t)

			if strings.Contains(line, "TX_FROM_BACK") && strings.TrimSpace(strings.Split(from[0], ">")[1]) == "1" {
				txts[float64(no)] = date
			} else if strings.Contains(line, "C_SIG_FROM_FRONT") && strings.TrimSpace(strings.Split(from[0], ">")[0]) == "1" {
				duration := txts[float64(no)].Sub(date).Abs().Seconds()
				pts = append(pts, plotter.XY{X: float64(no), Y: float64(duration)})
				data[no] = float64(duration)
			}
		}

	}
	temp := []float64{}
	for i := 1; i <= len(data); i++ {
		temp = append(temp, data[i])
		if i%100 == 0 {
			mean, _ := stats.Mean(temp)
			println(i/100, mean)
			temp = nil
		}
	}
	// println("mean", mean, "std", std)

	return pts
}
