module github.com/adely-rya/enigmap-go

go 1.26.1

require github.com/adely-rya/oram v0.0.0-20260724083517-09c11b3d4635

require (
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
)

replace github.com/adely-rya/oram => ../oram
