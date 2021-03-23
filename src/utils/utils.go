package utils

import (
	"fmt"
	"math/rand"
	"os"
)

func RandomInRange(low, hi int) int {
	return low + rand.Intn(hi-low)
}

func TokenGenerator(len int) string {
	b := make([]byte, len)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func CheckError(err error, function string, ignore bool) {
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Fatal error in %s: %s\n", function, err.Error())

		if !ignore {
			os.Exit(1)
		}
	}
}
