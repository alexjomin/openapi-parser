.PHONY: install
install:
	@echo "+ $@"
	go install

.PHONY: clean
clean:
	@echo "+ $@"
	rm -rf ./openapi-parser

openapi-parser:
	@echo "+ build $@"
	go build

DATAFILES?=user
.PHONY: datatest
datatest: openapi-parser
	@echo "+ $@ (DATAFILES=$(DATAFILES))"
	./openapi-parser --path datatest/$(DATAFILES)  --output datatest/$(DATAFILES)/openapi-generated.yaml
	diff datatest/$(DATAFILES)/openapi-valid.yaml datatest/$(DATAFILES)/openapi-generated.yaml

.PHONY: test
test: clean
	@echo "+ $@"
	@bash -c "trap '$(MAKE) clean;' EXIT; $(MAKE) .test"

.PHONY: .test
.test:
	@echo "+ $@"
	go test -v -race ./...
	$(MAKE) datatest DATAFILES=user
	$(MAKE) datatest DATAFILES=jsontags
	$(MAKE) datatest DATAFILES=jsonapitags
