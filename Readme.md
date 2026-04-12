# Docksmith – Minimal Container Runtime

Docksmith is a lightweight container runtime built in Go that implements core containerization concepts such as image manifests, layered filesystems, and namespace-based isolation.

---

##  Features

* Build container images from Docksmithfile
* Layered filesystem (content-addressable storage)
* Image manifest format (JSON)
* Environment variable support
* Working directory and command execution
* chroot + Linux namespaces for isolation
* Offline base image support (no network required)

---

##  Project Structure

```
docksmith/
├── cmd/
│   └── docksmith/
│       └── main.go          # Entry point (CLI)
├── internal/
│   ├── builder/            # Image build logic
│   ├── runtime/            # Container execution logic
│   ├── storage/            # Image + layer storage handling
│   └── model/              # Data structures (Image, Layer, Config)
├── utils/                  # Helper functions (tar, hashing, etc.)
├── Docksmithfile           # Build instructions (
```
