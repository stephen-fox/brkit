package pattern_test

import (
	"io"
	"log"
	"os"

	"gitlab.com/stephen-fox/brkit/pattern"
)

func ExampleDeBruijn_WriteToN() {
	db := &pattern.DeBruijn{}

	err := db.WriteToN(os.Stdout, 16)
	if err != nil {
		log.Fatalln(err)
	}
	os.Stdout.WriteString("\n")
	err = db.WriteToN(os.Stdout, 16)
	if err != nil {
		log.Fatalln(err)
	}
	os.Stdout.WriteString("\n")
	err = db.WriteToN(os.Stdout, 16)
	if err != nil {
		log.Fatalln(err)
	}

	// Output:
	// aaaabaaacaaadaaa
	// eaaafaaagaaahaaa
	// iaaajaaakaaalaaa
}

func ExampleDeBruijn_WriteToN_write_pattern_to_logger() {
	logger := log.New(os.Stdout, "", 0)

	db := &pattern.DeBruijn{
		OptLogger: logger,
	}

	err := db.WriteToN(io.Discard, 16)
	if err != nil {
		log.Fatalln(err)
	}
	err = db.WriteToN(io.Discard, 16)
	if err != nil {
		log.Fatalln(err)
	}
	err = db.WriteToN(io.Discard, 16)
	if err != nil {
		log.Fatalln(err)
	}

	// Output:
	// pattern string 0: aaaabaaacaaadaaa
	// pattern string 1: eaaafaaagaaahaaa
	// pattern string 2: iaaajaaakaaalaaa
}
