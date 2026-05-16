# Tasks: File Read & Delete APIs (004)

## Phase 1: Setup
*(No new project initialization needed; building on existing 003-upload-apis structure)*

## Phase 2: Foundational (Blocking Prerequisites)
- [ ] T001 [P] Update `BlobPresignDownloadInput` to include `Disposition` field in `file/model/blob_adapter_model.go`
- [ ] T002 [P] Implement `Disposition` parameter handling for S3 adapter in `file/service/blobAdapter/s3_compatible_adapter.go`
- [ ] T003 [P] Implement `Disposition` parameter handling for GCS adapter in `file/service/blobAdapter/gcs_adapter.go`
- [ ] T004 [P] Implement `Disposition` parameter handling for Azure adapter in `file/service/blobAdapter/azure_blob_adapter.go`
- [ ] T005 [P] Create read/delete specific constants in `file/utils/constant/read_delete_constants.go`
- [ ] T006 [P] Create read/delete specific error singletons in `file/error/file_read_delete_errors.go`
- [ ] T007 Extend `FileUploadRepository` interface with `FindManyScoped`, `FindVariantsByFileObjectIDs`, `FindVariantByCode`, and `DeleteFileObject` in `file/repository/file_repository.go`
- [ ] T008 Implement new repository methods in `file/repository/file_repository_impl.go`

## Phase 3: [US1] Seller Lists and Filters Uploaded Files (P1)
**Goal**: Allow sellers to retrieve a paginated, filtered list of their files.
**Independent Test**: Request files with various filters (purpose, status, mimeType) and verify tenant isolation and pagination.

- [ ] T009 [P] [US1] Create integration tests for batch list in `test/integration/file/get_all_files_test.go`
- [ ] T010 [P] [US1] Create input models (`GetFilesBase`, `GetFilesParam`, `GetFilesFilter`) and output models (`GetFilesResponse`, `FileItem`, `FileVariantItem`) in `file/model/file_read_delete_model.go`
- [ ] T011 [US1] Create `FileReadService` interface and implement `GetAllFiles` logic in `file/service/file_read_service.go`
- [ ] T012 [US1] Implement `GetAllFiles` HTTP handler in `file/handler/file_handler.go`
- [ ] T013 [US1] Register `GET /api/files` route in `file/route/file_operation_route.go`

## Phase 4: [US2] Seller Views a Single File's Details (P1)
**Goal**: Retrieve detailed metadata and an optional presigned download link for a specific file.
**Independent Test**: Request a single file with and without the `includeDownloadUrl` flag.

- [ ] T014 [P] [US2] Create integration tests for single get in `test/integration/file/get_file_test.go`
- [ ] T015 [P] [US2] Create output model `GetFileResponse` in `file/model/file_read_delete_model.go`
- [ ] T016 [US2] Implement `GetFile` logic in `file/service/file_read_service.go`
- [ ] T017 [US2] Implement `GetFile` HTTP handler in `file/handler/file_handler.go`
- [ ] T018 [US2] Register `GET /api/files/:fileId` route in `file/route/file_operation_route.go`

## Phase 5: [US5] Product Service Resolves File Data (P1)
**Goal**: Provide an in-process interface for the Product module to resolve file metadata.
**Independent Test**: Call `GetFilesByIDs` programmatically and verify cross-tenant files are returned successfully.

- [ ] T019 [P] [US5] Create integration tests for `GetFilesByIDs` in `test/integration/file/get_files_by_ids_test.go`
- [ ] T020 [US5] Implement `GetFilesByIDs` logic in `file/service/file_read_service.go`

## Phase 6: [US3] Seller Generates a Download Link for a File (P2)
**Goal**: Generate standalone short-lived download URLs for files and variants.
**Independent Test**: Generate URLs with different TTLs, variant codes, and dispositions.

- [ ] T021 [P] [US3] Create integration tests for download URL generation in `test/integration/file/download_url_test.go`
- [ ] T022 [P] [US3] Create output model `DownloadURLResponse` in `file/model/file_read_delete_model.go`
- [ ] T023 [US3] Implement `GetDownloadURL` logic in `file/service/file_read_service.go`
- [ ] T024 [US3] Implement `GetDownloadURL` HTTP handler with structured audit logging in `file/handler/file_handler.go`
- [ ] T025 [US3] Register `GET /api/files/:fileId/download-url` route in `file/route/file_operation_route.go`

## Phase 7: [US4] Seller Permanently Deletes a File (P2)
**Goal**: Synchronous hard-delete of storage blobs and database records.
**Independent Test**: Delete active and uploading files, verify blob removal and cascade deletion.

- [ ] T026 [P] [US4] Create integration tests for file deletion in `test/integration/file/delete_file_test.go`
- [ ] T027 [P] [US4] Create output model `DeleteFileResponse` in `file/model/file_read_delete_model.go`
- [ ] T028 [US4] Create `FileDeleteService` interface and implement `DeleteFile` logic in `file/service/file_delete_service.go`
- [ ] T029 [US4] Implement `DeleteFile` HTTP handler with structured audit logging in `file/handler/file_handler.go`
- [ ] T030 [US4] Register `DELETE /api/files/:fileId` route in `file/route/file_operation_route.go`

## Phase 8: Polish & Cross-Cutting Concerns
- [ ] T031 Wire `FileReadService` and `FileDeleteService` singletons in `file/factory/singleton/service_factory.go`
- [ ] T032 Expose new services in `file/factory/singleton/singleton_factory.go`
- [ ] T033 Inject services into `FileHandler` in `file/factory/singleton/handler_factory.go`
- [ ] T034 [P] Create performance benchmarking test to validate p95 latency SLOs (SC-002 to SC-005) in `test/integration/file/performance_test.go`

---

## Execution Dependencies
1. Phase 2 (Foundational) MUST be completed before any User Story phase.
2. US1, US2, and US5 can be implemented in parallel after Phase 2.
3. US3 and US4 can be implemented in parallel after Phase 2, but preferably after US2 for shared `file_read_service.go` structure.
4. Phase 8 (Polish) MUST be completed after all User Stories to correctly wire the factories.

## Parallel Execution Examples
- Developer A works on extending `blob_adapter_model.go` and updating adapters (T001-T004).
- Developer B works on extending `file_repository.go` and `file_repository_impl.go` (T007-T008).
- Developer A writes integration tests for US1 (T009) while Developer B builds the models for US1 (T010).
