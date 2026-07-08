.PHONY: test verify-dependency-security

test:
	go test ./...

verify-dependency-security:
	bash scripts/verify-dependency-security.sh
