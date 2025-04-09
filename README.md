# arch

[![Go Reference](https://pkg.go.dev/badge/golang.org/x/arch.svg)](https://pkg.go.dev/golang.org/x/arch)

This repository holds machine architecture information used by the Go toolchain.
The parts needed in the main Go repository are copied in.

## Report Issues / Send Patches

This repository uses Gerrit for code changes. To learn how to submit changes to
this repository, see https://go.dev/doc/contribute.

The git repository is https://go.googlesource.com/arch.

The main issue tracker for the arch repository is located at
https://go.dev/issues. Prefix your issue with "x/arch:" in the
subject line, so it is easy to find.

Steps to build the changes:
1) Make changes to the files in "arch/s390x" repo...
2) Do a "git push" of local branch changes to here.. git push git@github.com:Vishwanatha-HD/arch.git...
3) Edit the "src/cmd/internal/disasm/disasm.go" file to add "github.com:Vishwanatha-HD/arch/s390x/s390xasm" as an import path..
4) From "src/cmd" directory, get the latest changes of the github.com/arch repo by executing "go get github.com/Vishwanatha-HD/arch@<latest_commit_id>" cmd...
5) Execute "go mod tidy" command from "src/cmd" directory.. This will update the go.mod file with the github.com/arch repo with the latest commit hash..
6) Execute "go mod vendor" command from "src/cmd/objdump" directory...
7) Execute "go clean" and "go build" commands..
8) This will create a "objdump" binary inside "src/cmd/objdump" directory..
9) Execute "go clean", "go build" and "go test -c" commands from "arch/s390x/s390xasm" directory to build the local changes & create a s390xasm.test file.. 
10) Use the "objdump" binary which is locally built inside "src/cmd/objdump" to disassemble the s390xasm.test file.. For eg..
11) Execute the following command.. "/home/vishwa/golang/go/src/cmd/objdump/objdump -gnu s390xasm.test > <txt_file>"
12) The new "txt_file" created will have the latest changes done to "arch/s390x/s390xasm" directory...
