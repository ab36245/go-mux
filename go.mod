module github.com/ab36245/go-mux

go 1.24.4

replace github.com/ab36245/go-websocket => ../go-websocket

replace github.com/ab36245/go-writer => ../go-writer

replace github.com/ab36245/go-errors => ../go-errors

require (
	github.com/ab36245/go-websocket v0.0.0-20250626045342-b297777d70a1
	github.com/rs/zerolog v1.34.0
)

require (
	github.com/ab36245/go-errors v0.0.0-20250428061939-8b056c3b905e // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.33.0 // indirect
)
