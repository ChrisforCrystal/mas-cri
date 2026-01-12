# Tasks: Feature 005 Native Runtime

- [ ] T001 Spec & Plan: Create documents (Done)
- [ ] T002 Environment: Check Linux environment & Kernel capabilities
- [ ] T003 Dependencies: `go get github.com/opencontainers/runc`
- [ ] T004 Infrastructure: Create `pkg/native` package and `NativeAdapter` struct
- [ ] T005 Rootfs: Implement simple `SetupRootfs` (extract tar to dir)
- [ ] T006 Libcontainer: Implement `CreateContainer` using `libcontainer.Config`
- [ ] T007 Libcontainer: Implement `StartContainer`
- [ ] T008 Libcontainer: Implement `Stop` and `Remove`
- [ ] T009 Switch: Update `main.go` to support switching between Docker/Native backends
- [ ] T010 Verification: Run a container on Lima without Docker
