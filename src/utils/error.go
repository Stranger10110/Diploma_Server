package utils

import (
	"fmt"
	"os"
)

func CheckError(err error, function string, ignore bool) {
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Fatal error in %s: %s\n", function, err.Error())

		if !ignore {
			os.Exit(1)
		}
	}
}
