# Phase A Baseline (Completed)

Date: 2026-03-04  
Checkpoint tag: `phase-a-baseline-ec4563d`

## Scope

Phase A goals were:

1. Freeze a rollback checkpoint.
2. Build a compile baseline.
3. Confirm MySQL role.
4. Confirm filesystem role.
5. Confirm Qdrant role.
6. Confirm PageIndex role.

## Results

### 0) Module Path Migration

- Replaced internal imports from `GopherAI/...` to:
  - `github.com/nanjiek/GopherAI_HybridAgent/...`
- Updated `go.mod` module line to:
  - `module github.com/nanjiek/GopherAI_HybridAgent`
- Executed `go mod tidy` successfully.

### 1) Checkpoint

- Created local tag: `phase-a-baseline-ec4563d`

### 2) Compile Baseline

- Full `go test ./...` currently fails on image pipeline due to `onnxruntime_go` build constraints in this environment.
- Non-image backend packages compile successfully:
  - `common/*` (except image), `config`, `controller/*` (except image), `dao/*`, `middleware/*`, `model`, `service/*` (except image), `utils/*`.

### 3) MySQL Decision

- Keep MySQL as business/metadata source:
  - users, sessions, messages
  - news source metadata
  - article/event relations
  - index status/version metadata

### 4) Filesystem Decision

- Use filesystem as source of truth for long-form content:
  - cleaned documents
  - page files
  - news article normalized text

Suggested layout:

- `data/docs/{source}/{doc_id}/raw.*`
- `data/docs/{source}/{doc_id}/clean.md`
- `data/docs/{source}/{doc_id}/pages/page_{n}.md`

### 5) Qdrant Decision

- Qdrant is vector retrieval infrastructure only (not business truth).
- Store vector payload with strict references:
  - `doc_id`, `page_id`, `chunk_id`, `path`, `source`, `published_at`.

### 6) PageIndex Decision

- PageIndex is primary retrieval layer for long documents.
- Vector retrieval is used as semantic complement, then reranking.

## Phase B Input

Phase B can start with schema/protocol definition for:

1. ID conventions (`doc_id/page_id/chunk_id`)
2. MySQL table contracts
3. PageIndex JSON schema
4. Qdrant collection/payload schema
5. unified retrieval response contract
