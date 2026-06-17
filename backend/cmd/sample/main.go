package main

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

type sample struct {
	chekc bool `validate:"required"`
}

func main() {

	data := &sample{
		chekc: false,
	}

	v := validator.New()

	if err := v.Struct(data); err != nil {
		fmt.Println(err)
	} else {
		println("validation passed")
	}
}
