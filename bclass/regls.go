package bclass

import (
	"fmt"
	"github.com/gonum/matrix/mat64"
)

type Model struct {
	W      mat64.Dense
	Deg    int
	Lambda float64
}

type GlobalModel struct {
	ModelList map[int]Model
	TestSize  map[int]int
	D         int
}

func (model Model) Print() {
	temp := &model.W
	w := mat64.Formatted(temp, mat64.Prefix("    "))
	fmt.Printf("Model Weights:\nw = %v\n\n", w)
}

func (model Model) Predict(xt *mat64.Dense) *mat64.Dense {
	xpoly := PolyBasis(xt, xt, 0, model.Deg)

	r, _ := xt.Dims()
	_, c := model.W.Dims()
	yt := mat64.NewDense(r, c, nil)
	yt.Mul(xpoly, &model.W)
	tresh(yt)

	return yt
}

func (model GlobalModel) Predict(xt *mat64.Dense) *mat64.Dense {
	r, _ := xt.Dims()
	agg := mat64.NewDense(r, 1, nil)
	for k, m := range model.ModelList {
		temp := m.Predict(xt)
		temp.Scale(float64(model.TestSize[k])/float64(model.D), temp)
		agg.Add(agg, temp)
	}
	tresh(agg)

	return agg
}

func RegLSBasisC(x, y *mat64.Dense, lambda float64, deg int) Model {
	xpoly := PolyBasis(x, x, 0, deg)

	r, c := xpoly.Dims()
	xtx := mat64.NewDense(c, c, nil)
	eye := Eye(c)
	xtx.Mul(xpoly.T(), xpoly)
	eye.Scale(lambda, eye)
	xtx.Add(xtx, eye)

	w := mat64.NewDense(c, 1, nil)
	k := mat64.NewDense(c, r, nil)
	k.Solve(xtx, xpoly.T())
	w.Mul(k, y)

	model := Model{*w, deg, lambda}

	return model
}

func PolyBasis(xpoly, x *mat64.Dense, ind, deg int) *mat64.Dense {
	r, c := xpoly.Dims()
	t := mat64.NewDense(r, c, nil)
	t.Copy(xpoly)

	if ind == deg {
		if deg != 1 {
			t.MulElem(t, x)
		}
		return t
	} else if ind == 0 {
		data := make([]float64, r)
		for i := range data {
			data[i] = 1.0
		}
		t1 := mat64.NewDense(r, 1, data)
		t2 := PolyBasis(t, x, ind+1, deg)
		x2 := mat64.NewDense(r, (deg-ind)*c+1, nil)
		x2.Augment(t1, t2)
		return x2
	} else if ind == 1 {
		t2 := PolyBasis(t, x, ind+1, deg)
		x2 := mat64.NewDense(r, (deg-ind+1)*c, nil)
		x2.Augment(x, t2)
		return x2
	} else {
		t.MulElem(t, x)
		t2 := PolyBasis(t, x, ind+1, deg)
		x2 := mat64.NewDense(r, (deg-ind+1)*c, nil)
		x2.Augment(t, t2)
		return x2
	}
}

func Eye(d int) *mat64.Dense {
	eye := mat64.NewDense(d, d, nil)
	for i := 0; i < d; i++ {
		eye.Set(i, i, 1.0)
	}
	return eye
}

func TestResults(predict, test *mat64.Dense) (c, d int) {
	d, _ = predict.Dims()
	c = 0
	for i := 0; i < d; i++ {
		if (predict.At(i, 0) * test.At(i, 0)) > 0 {
			c++
		}
	}
	return c, d
}

func tresh(v *mat64.Dense) {
	r, _ := v.Dims()
	for i := 0; i < r; i++ {
		if v.At(i, 0) < 0 {
			v.Set(i, 0, -1.0)
		} else {
			v.Set(i, 0, 1.0)
		}
	}
}
