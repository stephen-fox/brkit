module gitlab.com/stephen-fox/brkit/cmd/dasm

go 1.16

require (
	gitlab.com/stephen-fox/brkit v0.3.0
	gitlab.com/stephen-fox/brkit/asmkit v0.3.0
	golang.org/x/arch v0.14.0
)

replace gitlab.com/stephen-fox/brkit/asmkit => ../../asmkit
