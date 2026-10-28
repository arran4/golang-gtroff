package groff

//go:generate go run github.com/arran4/go-subcommand/cmd/gosubc generate --force --timestamp=false --replace-template templates
//go:generate go run ./cmd/gen
//go:generate go run ./cmd/examples
//go:generate go run ./cmd/docgen --write
