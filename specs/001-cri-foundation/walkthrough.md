# Walkthrough: Basic MasCRI Foundation (Feature 001)

## What was built

We established the "Hello World" of CRI:

1.  **gRPC Server**: A functional CRI shim listening on Unix Socket.
2.  **Trace Interceptor**: A middleware that logs every request in JSON.
3.  **Mock Implementations**: `Version` works, `RunPodSandbox` returns a fake ID.

## Verification Steps

### 1. Prerequisites

Ensure `crictl` is installed. If not using Homebrew:

```bash
brew install cri-tools
```

### 2. Start the Server

```bash
make run
```

_Output_: `MasCRI listening on /tmp/mascri.sock`

### 3. Check Version (Client)

In a new terminal:

```bash
make verify-info
```

_Output_:

```json
{
  "status": {
    "conditions": [
      {
        "type": "RuntimeReady",
        "status": true,
        "message": "MasCRI is ready to rock"
      }
    ]
  }
}
```

### 4. Create a Pod Sandbox (Simulated)

```bash
make verify-runp
```

_Output_: `sandbox-fake-12345`

### 5. Observe the Server Logs (The "Aha!" Moment)

Back in the server terminal, look at the logs. You will see the **full Pod configuration** sent by the client:

```text
INFO[..] --> [gRPC Request] method=/runtime.v1.RuntimeService/RunPodSandbox body="{\"config\":{\"metadata\":{\"name\":\"nginx-sandbox\",\"uid\":\"1\",\"namespace\":\"default\"}}}"
```

This proves our "Traceability" principle is working. We can "see" what Kubelet is asking for.
