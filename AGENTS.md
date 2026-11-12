# bam

Follow [the gatework Go engineering standard](https://github.com/gatework/go-standards/blob/v1.0.2/STANDARD.md) (go-standards/STANDARD.md); this file adds the BAM-specific contracts. Run `make check` before committing.

- Keep the runtime standard-library-only. The root package owns HTTP, authentication, pagination and lossless JSON; SDK adapters select typed capabilities. [ADR 0003](docs/adr/0003-sdk-composition.md) defines their dependency layers, enforced by `architecture.yaml`.
- Public handles keep their implementation fields private and declare their supported methods explicitly. Read DTOs expose named fields; immutable query values are built with constructors and reject an unset value before I/O.
- For domain changes, consult [the glossary](GLOSSARY.md), [ADR 0002](docs/adr/0002-typed-domain-capabilities.md) and [capability evidence](docs/rest-v2-coverage.md). Preserve unknown JSON fields and distinguish unchanged, zero and null values.
- Keep deployment submission explicit and writes free of automatic retries. Preserve same-origin pagination scope, bounded responses, TLS verification and per-client credential ownership. Credentials and appliance data stay out of tests and logs.
- Exercise public HTTP contracts with synthetic `httptest` servers. Local contract tests and CI do not establish live-appliance acceptance.
- Update capability evidence when supported operations change. Release through the existing tag workflow using the successful exact-main-commit CI source artifact; verify downloaded assets against that artifact.
