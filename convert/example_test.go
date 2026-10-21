package convert_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/arran4/golang-gtroff/convert"
)

func ExampleHTMLToMan() {
	_, err := convert.HTMLToMan(strings.NewReader("<p>.danger</p>"), os.Stdout, convert.Options{Strict: true})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// .PP
	// \&\f[R].danger\f[R]\c
}
