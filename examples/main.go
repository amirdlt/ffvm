package main

import (
	"github.com/amirdlt/ffvm"
	"github.com/amirdlt/flex/util"
)

type Something struct {
	FirstName    string `ffvm:"lower,required;upper;min_len=1;len=100"`
	Age          any    `ffvm:"upper,max=5;max=11"`
	AnotherThing *AnotherThing
}

type AnotherThing struct {
	SecondName    string `ffvm:"lower,regex=dlt" json:"second_name,omitempty"`
	AnotherThings []*AnotherThing
}

func main() {
	st := Something{
		FirstName: "Amir",
		Age:       12,
		AnotherThing: &AnotherThing{
			SecondName: "Hassan",
			AnotherThings: []*AnotherThing{
				{
					SecondName: "Abbas",
					AnotherThings: []*AnotherThing{
						{
							SecondName:    "Majid",
							AnotherThings: nil,
						},
						{
							SecondName:    "Name1",
							AnotherThings: nil,
						},
						{
							SecondName:    "Name2",
							AnotherThings: nil,
						},
						{
							SecondName:    "Name3",
							AnotherThings: nil,
						},
						{
							SecondName:    "dlt",
							AnotherThings: nil,
						},
						{
							SecondName: "Majid2",
							AnotherThings: []*AnotherThing{
								{
									SecondName:    "What",
									AnotherThings: nil,
								},
							},
						},
					},
				},
			},
		},
	}

	//r, _ := util.GetFileStream("./log.txt")
	for index, issue := range ffvm.Validate(&st) {
		//fmt.Fprintln(r, issue.Field)
		util.Println(index, issue.Field)
	}
}
