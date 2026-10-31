module github.com/bencode/jun

go 1.27.3

require github.com/egoist/mygo v0.7.3

require (
	github.com/ebitengine/purego v0.11.1 // indirect
	github.com/go-text/typesetting v0.3.5 // indirect
	golang.org/x/image v0.46.0 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/text v0.43.0 // indirect
)

tool github.com/egoist/mygo/cmd/mygo

replace github.com/egoist/mygo => github.com/bencode/mygo v0.7.4-0.20261030154123-e78dff14b2cb
