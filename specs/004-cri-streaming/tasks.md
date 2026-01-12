# Tasks: Feature 004 Streaming IO

- [ ] T001 Spec & Plan: Create documents (Done)
- [ ] T002 Dependencies: Update `go.mod` for `k8s.io/kubelet` (if needed for streaming)
- [ ] T003 Infrastructure: Create `pkg/stream` and implement `Runtime` interface stubs
- [ ] T004 Server: Initialize and Start `streaming.Server` in `MasCRIServer`
- [ ] T005 Logs: Implement `ReopenContainerLog` (Stub mostly) and `docker logs` via Adapter
- [ ] T006 Exec: Implement `Exec` RPC (Return URL)
- [ ] T007 Exec: Implement `Exec` Backend (Call Docker Adapter)
- [ ] T008 Adapter: Add `ExecInContainer` using simple `docker exec` (non-interactive first)
- [ ] T009 Adapter: Upgrade `ExecInContainer` to support Interactive Stream (Attach) - _Hard_
- [ ] T010 Verification: Verify `crictl logs`
- [ ] T011 Verification: Verify `crictl exec`
