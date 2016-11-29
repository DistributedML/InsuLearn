package main

import (
	"fmt"
	"github.com/4180122/distbayes/bclass"
	"github.com/gonum/matrix/mat64"
	"io/ioutil"
	"strconv"
	"strings"
)

func main() {

	// // initialize a 3 element float64 slice
	// dx := make([]float64, 6)
	// dy := make([]float64, 3)
	// dt := make([]float64, 2)
	// dv := make([]float64, 1)

	// // set the elements
	// dx[0] = -0.8920
	// dx[1] = 0.5024
	// dx[2] = 0.9963
	// dx[3] = -0.5948
	// dx[4] = -0.9135
	// dx[5] = -0.8960

	// dy[0] = -1
	// dy[1] = 1
	// dy[2] = -1

	// dt[0] = -0.7481
	// dt[1] = 0.9425

	// dv[0] = -1

	// pass the slice dx as data to the matrix x
	// x := mat64.NewDense(3, 2, dx)
	// y := mat64.NewDense(3, 1, dy)
	// t := mat64.NewDense(1, 2, dt)
	// v := mat64.NewDense(1, 1, dv)

	xg := ReadData("x.txt")
	yg := ReadData("y.txt")

	modlist := make(map[int]bclass.Model)
	X := make(map[int]*mat64.Dense)
	Y := make(map[int]*mat64.Dense)
	C := make(map[int]int)

	X[1] = ReadData("x1.txt")
	X[2] = ReadData("x2.txt")
	X[3] = ReadData("x3.txt")
	Y[1] = ReadData("y1.txt")
	Y[2] = ReadData("y2.txt")
	Y[3] = ReadData("y3.txt")

	// v_hat1 := model1.Predict(t)
	// v_hat2 := model2.Predict(t)
	// v_hat3 := model2.Predict(t)
	// c1, d1 := bclass.TestResults(v_hat1, v)
	// c2, d2 := bclass.TestResults(v_hat2, v)
	// fmt.Println(float64(c1)/float64(d1), float64(c2)/float64(d2))

	for i := range X {
		modlist[i] = bclass.RegLSBasisC(X[i], Y[i], 0.01, 2)
		modlist[i].Print()
	}

	// modelg := bclass.GlobalModel{modlist, tstlist, 0}

	// modelg.ModelList[1] = model1
	// modelg.ModelList[2] = model2
	// modelg.TestSize[1] = c1
	// modelg.TestSize[2] = c2
	// modelg.D = d1 + d2

	dmax := 0
	for k, model := range modlist {
		C[k] = 0
		dtemp := 0
		for ktest := range modlist {
			y := model.Predict(X[ktest])
			c, d := bclass.TestResults(y, Y[ktest])
			C[k] = C[k] + c
			dtemp = dtemp + d
		}
		if dtemp > dmax {
			dmax = dtemp
		}
	}

	modelg := bclass.GlobalModel{modlist, C, dmax}

	// v_hatg := modelg.Predict(t)
	// cg, dg := bclass.TestResults(v_hatg, v)

	yhat := modelg.Predict(xg)
	cg, dg := bclass.TestResults(yhat, yg)

	//w := mat64.Formatted(Y[1], mat64.Prefix("    "))
	//fmt.Printf("X_1:\nw = %v\n\n", w)

	modtot := bclass.RegLSBasisC(xg, yg, 0.01, 2)
	yhatot := modtot.Predict(xg)
	ct, dt := bclass.TestResults(yhatot, yg)

	fmt.Println(float64(cg)/float64(dg), float64(ct)/float64(dt))

}

func ReadData(filename string) *mat64.Dense {
	dat, _ := ioutil.ReadFile(filename)
	array := strings.Split(string(dat), "\n")
	r := len(array) - 1
	temp := strings.Split(array[0], ",")
	c := len(temp)
	vdat := make([]float64, c*r)
	for i := 0; i < r; i++ {
		temp = strings.Split(array[i], ",")
		for j := 0; j < c; j++ {
			vdat[i*c+j], _ = strconv.ParseFloat(temp[j], 64)
		}
	}
	return mat64.NewDense(r, c, vdat)
}
