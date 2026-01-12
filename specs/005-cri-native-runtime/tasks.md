# Tasks: Feature 005 Native Runtime

- [x] T001 Spec & Plan: Create documents (Done)
- [x] T002 Environment: Check Linux environment & Kernel capabilities (Done via Lima)
- [x] T003 Dependencies: `go get github.com/opencontainers/runc`
- [x] T004 Infrastructure: Create `pkg/native` package and `NativeAdapter` struct
- [x] T005 Rootfs: Implement simple `SetupRootfs` (extract tar to dir)
- [x] T006 Libcontainer: Implement `CreateContainer` using `libcontainer.Config`
- [x] T007 Libcontainer: Implement `StartContainer`
- [x] T008 Libcontainer: Implement `Stop` and `Remove`
- [x] T009 Switch: Update `main.go` to support switching between Docker/Native backends
- [ ] T010 Verification: Run a container on Lima without Docker
