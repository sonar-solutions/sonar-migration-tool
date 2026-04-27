module github.com/sonar-solutions/sonar-migration-tool

go 1.25.0

require (
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/go-pdf/fpdf v0.9.0
	github.com/sonar-solutions/sq-api-go v0.0.0
	github.com/spf13/cobra v1.8.1
	golang.org/x/sync v0.20.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/mattn/go-colorable v0.1.2 // indirect
	github.com/mattn/go-isatty v0.0.8 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/term v0.42.0 // indirect
	golang.org/x/text v0.36.0 // indirect
)

exclude github.com/yuin/goldmark v1.4.13

replace github.com/sonar-solutions/sq-api-go => ../lib/sq-api-go
