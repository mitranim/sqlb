MAKEFLAGS := --silent --always-make
TESTFLAGS := $(if $(filter $(verb), true), -v,) -count=1
TEST      := test $(TESTFLAGS) -timeout=1s -run=$(run)
BENCH     := test $(TESTFLAGS) -run=- -bench=$(or $(run),.) -benchmem -benchtime=128ms
WATCH     := watchexec -r -c -d=0 -n

test_w:
	gow -c -v $(TEST)

test:
	go $(TEST)

bench_w:
	gow -c -v $(BENCH)

bench:
	go $(BENCH)

lint_w:
	$(WATCH) -- $(MAKE) lint

lint:
	golangci-lint run
	echo [lint] ok
