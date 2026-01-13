package utils_test

import (
	"fmt"
	"testing"

	"github.com/lwy110193/go_vendor/utils"
)

type AA struct {
	FieldOne string
	FieldTwo int
}

type TestStruct struct {
	FieldFdsdf  string `json:"field1"`
	FieldGavdds int    `json:"field2"`
	Aobj        *AA    `json:"aobj"`
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

func TestGetAttrValueList(t *testing.T) {
	d := []TestStruct{
		{
			FieldFdsdf:  "value1",
			FieldGavdds: 42,
			Aobj: &AA{
				FieldOne: "a1",
				FieldTwo: 1,
			},
		},
		{
			FieldFdsdf:  "value2",
			FieldGavdds: 43,
			Aobj: &AA{
				FieldOne: "a2",
				FieldTwo: 2,
			},
		},
	}
	result := []*AA{}
	utils.GetAttrValueList(d, "Aobj", &result)
	fmt.Printf("%#v", result)
}
