package utils_test

import (
	"fmt"
	"testing"

	"github.com/lwy110193/go_vendor/utils"
)

type TestStruct struct {
	FieldFdsdf  string `json:"field1"`
	FieldGavdds int    `json:"field2"`
}
type Result struct {
	FieldOne string
	FieldTwo int
	TestStruct
}

func TestConvStructToMap(t *testing.T) {
	d := Result{
		FieldOne: "value1",
		FieldTwo: 42,
		TestStruct: TestStruct{
			FieldFdsdf:  "value1",
			FieldGavdds: 42,
		},
	}
	m := utils.MI{}
	utils.ConvStructToMap(d, m)
	fmt.Printf("%#v", m)
}
