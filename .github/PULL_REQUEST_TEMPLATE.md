## Summary

<!-- One paragraph: what changed and why. -->

## Type of change

<!-- Put x in [ ] for the relevant boxes. -->

- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would change the public API or behavior)
- [ ] Documentation only (no code change)
- [ ] Refactor / cleanup (no behavior change)

## Checklist

<!-- Items below are required for the PR to be reviewed. -->

- [ ] `gofmt -l .` produces no output
- [ ] `go vet ./...` is clean
- [ ] `go test -race ./...` passes
- [ ] `golangci-lint run` produces 0 issues
- [ ] `govulncheck ./...` reports 0 vulnerabilities
- [ ] `go fix ./...` produces no diff
- [ ] I added/updated tests for the change
- [ ] I updated relevant docs (README, recipes, references, ADRs)
- [ ] If the public API changed, I added/changed the appropriate ADR in `docs/adr/`
- [ ] If I added a new dependency, I justified why stdlib is not enough (per ADR-0001)
- [ ] I did NOT add secrets, API keys, or internal URLs

## References

- Closes #… (if any)
- Discussed in #… (if any)
