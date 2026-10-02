module github.com/manny-ovena/microendpoint-stack/services/endpoints/inventory.adjust

go 1.27.0

require github.com/manny-ovena/microendpoint-stack/shared/logging v0.0.0

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/rs/zerolog v1.33.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
)

replace github.com/manny-ovena/microendpoint-stack/shared/logging => ../../../shared/logging
