exec: *.go */*.go go.*
	# the executable
	go build -o $@ -ldflags "-s -w"
	file $@

module.tar.gz: exec
	# the bundled module
	rm -f $@
	tar czf $@ $^