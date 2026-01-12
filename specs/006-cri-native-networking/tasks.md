# Feature 006: Native Runtime CNI Networking

## Tasks

- [x] Inspect existing CNI integration in `pkg/network` and `DockerAdapter`
- [x] Implement `GetNetNS` correctly in `pkg/native/adapter.go`
- [x] Update `RunPodSandbox` in `pkg/native/adapter.go` to invoke CNI `SetUp`
- [x] Update `StopPodSandbox` in `pkg/native/adapter.go` to invoke CNI `TearDown`
- [ ] Verify networking setup by inspecting namespace configuration (Blocked by Runtime Environment)
