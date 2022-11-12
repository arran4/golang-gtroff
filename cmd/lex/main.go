package main

import (
	"bufio"
	"fmt"
	groff "github.com/arran4/golang-gtroff"
	"io"
	"log"
	"os"
)

func main() {
	for i, f := range os.Args {
		if i == 0 {
			continue
		}
		log.Printf("File %s", f)
		tokens, err := ParseFile(f)
		if err != nil {
			log.Panicln(err)
		}
		for _, token := range tokens {
			log.Printf("%#v", token)
		}
	}
}

func ParseFile(fn string) ([]groff.Token, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, fmt.Errorf("open file: %s: %w", fn, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Panicln("Close error", err)
		}
	}()
	var tokens []groff.Token
	br := bufio.NewReader(f)
	for i := 0; ; i++ {
		line, err := br.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read line: %s:%d: %w", fn, i, err)
		}
		token, err := groff.ParseLine(line)
		if err != nil {
			return nil, fmt.Errorf("parse line: %s:%d: %w", fn, i, err)
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}
