# Tasks: Feature 003 CNI Networking

- [ ] T001 Spec & Plan: Create documents (Done)
- [x] T002 Dependencies: `go get github.com/containernetworking/cni`
- [x] T003 Infrastructure: Create `pkg/cni` and `CNIManager` struct
- [x] T004 Implementation: Implement `SetUpPod` (CNI ADD)
- [x] T005 Implementation: Implement `TearDownPod` (CNI DEL)
- [x] T006 Adapter Update: Add `GetNetNS(containerID)` to Docker Adapter (parsing `SandboxKey`)
- [x] T007 Integration: Wire `CNIManager` into `MasCRIServer`
- [x] T008 Integration: Update `RunPodSandbox` to call `SetUpPod`
- [x] T009 Integration: Update `StopPodSandbox` to call `TearDownPod`
- [x] T010 Verification: Install CNI plugins & Config locally
- [x] T011 Verification: Run `crictl runp` and check logs/IP
