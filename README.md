To make go deps
"go mod init <projname.go> " and "go mod tidy"  needs to be ran
this generates a go.mod file which has dependencies and a go.sum which stores the versions of the dependencies
"go mod download" downloads dependencies on any system. These go programs can be compiled into native binaries for the target system and architecture.
