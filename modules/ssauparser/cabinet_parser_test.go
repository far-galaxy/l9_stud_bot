package ssauparser

import (
	"testing"
)

func TestLoadJSON(t *testing.T) {
	q := Query{
		YearID:  9,
		Week:    17,
		GroupID: 530996168,
	}

	d, err := LoadJSON(q)
	if err != nil {
		t.Fatal(err)
	}
	ParseData(q, d)
}
