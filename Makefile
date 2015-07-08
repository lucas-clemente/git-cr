.PHONY: release

release:
	GOOS=darwin go build
	zip git-cr.osx.zip git-cr
	GOOS=linux go build
	zip git-cr.linux.zip git-cr
