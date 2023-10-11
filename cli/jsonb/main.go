package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/Harazi/jsonb"
)

func main() {
	input := flag.String("i", "-", "`input` file")
	output := flag.String("o", "-", "`output` file")
	encode := flag.Bool("e", false, "Eecode json into jsonb")
	decode := flag.Bool("d", false, "Decode jsonb into json")
	flag.Parse()

	if *encode == *decode {
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "Either -e or -d flag arguments must be set")
		os.Exit(1)
	}

	var in, out []byte
	var err error

	if *input == "-" {
		in, err = io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		in, err = os.ReadFile(*input)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *encode {
		out, err = jsonb.Encode(string(in))
		if err != nil {
			log.Fatal(err)
		}
	} else {
		outStr, err := jsonb.Decode(in)
		if err != nil {
			log.Fatal(err)
		}
		out = []byte(outStr)
	}

	if *output == "-" {
		fmt.Printf("%s", out)
	} else {
		err := os.WriteFile(*output, out, 0644)
		if err != nil {
			log.Fatal(err)
		}
	}
}
