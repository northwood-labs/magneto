---
inclusion: manual
---

# Adversarial Review Architecture Constraints

## Blast-radius domains

The following domains trigger mandatory adversarial review when an artifact touches them:

* auth — Authentication and authorization flows
* secrets — Secret management, key rotation, credential storage
* payments — Financial transactions, billing, refund logic
* data-integrity — Data migration, schema changes, consistency guarantees
* irreversible-actions — Deletion flows, account termination, data purging

## Foundational trust artifacts

Artifacts consumed by downstream automation without independent verification:

* Design documents consumed by task generation
* Architecture decision records consumed by implementation agents
* Configuration schemas consumed by deployment pipelines
