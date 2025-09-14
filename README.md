# FileOps - Advanced File Operations Tool

[![Go Report Card](https://goreportcard.com/badge/github.com/a4abhishek/fileops)](https://goreportcard.com/report/github.com/a4abhishek/fileops)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/release/a4abhishek/fileops.svg)](https://github.com/a4abhishek/fileops/releases)

A high-performance, AI-powered file operations toolkit for advanced file management, deduplication, and intelligent organization.

## ✨ Features

### Core Operations
- 🧹 **Smart Cleanup**: Remove empty directories recursively with safety checks
- 📦 **File Consolidation**: Move and organize files with multiple strategies
- 🔍 **Advanced Deduplication**: Lightning-fast duplicate detection using optimized algorithms
- 🖼️ **Image Similarity**: AI-powered detection of similar/cropped images
- 🤖 **Intelligent Organization**: ML-based automatic file organization
- ⚡ **Pipeline Support**: Chain operations for complex workflows

### Performance Features
- 🚀 **Multi-core Processing**: Leverage all available CPU cores
- 💾 **Memory Efficient**: Streaming processing for large datasets
- ⚡ **SIMD Acceleration**: Optimized hash algorithms
- 📊 **Progress Tracking**: Real-time progress with ETA
- 🔄 **Resume Operations**: Continue interrupted operations

### Safety & Reliability
- 🛡️ **Dry Run Mode**: Preview changes before execution
- 🔒 **Safe Operations**: Atomic operations with rollback capability
- 📝 **Comprehensive Logging**: Detailed operation logs
- ✅ **Validation**: Pre-flight checks and validation

## 🚀 Quick Start

### Installation

#### Download Pre-built Binary
```bash
# Linux/macOS
curl -sfL https://raw.githubusercontent.com/a4abhishek/fileops/main/install.sh | sh

# Windows (PowerShell)
iwr -useb https://raw.githubusercontent.com/a4abhishek/fileops/main/install.ps1 | iex
```

#### Build from Source
```bash
git clone https://github.com/a4abhishek/fileops.git
cd fileops
go build -o fileops ./cmd/fileops
```

### Basic Usage

```bash
# Clean empty directories
fileops clean /path/to/directory --dry-run

# Deduplicate files
fileops dedup /path/to/files --algorithm blake2b

# Consolidate files
fileops consolidate /source1 /source2 --dest /target --strategy date

# Find similar images
fileops similar-images /photos --threshold 0.85

# AI-powered organization
fileops organize /unsorted --ai-organize

# Run a pipeline
fileops pipeline run cleanup-and-organize.yaml
```

## 📖 Documentation

- [Complete Documentation](https://github.com/a4abhishek/fileops/wiki)
- [API Reference](https://github.com/a4abhishek/fileops/wiki/API-Reference)
- [Configuration Guide](https://github.com/a4abhishek/fileops/wiki/Configuration)
- [Pipeline Examples](https://github.com/a4abhishek/fileops/wiki/Pipeline-Examples)
- [Performance Tuning](https://github.com/a4abhishek/fileops/wiki/Performance-Tuning)

## 🛠️ Development

### Prerequisites
- Go 1.21+
- Python 3.9+ (for ML features)
- Node.js 18+ (for web UI)

### Setup Development Environment
```bash
git clone https://github.com/a4abhishek/fileops.git
cd fileops

# Install Go dependencies
go mod download

# Setup Python ML environment
cd ml-service
pip install -r requirements.txt

# Setup web UI
cd ../web-ui
npm install
```

### Project Structure
```
fileops/
├── cmd/                    # CLI entry points
├── internal/               # Private application code
│   ├── engine/            # Core operation engines
│   ├── api/               # REST API server
│   ├── pipeline/          # Pipeline orchestration
│   └── ml/                # ML integration
├── ml-service/            # Python ML microservice
├── web-ui/                # React web interface
├── pkg/                   # Public libraries
├── configs/               # Configuration files
├── docs/                  # Documentation
└── examples/              # Example configurations
```

### Running Tests
```bash
# Unit tests
go test ./...

# Integration tests
go test -tags=integration ./...

# Benchmarks
go test -bench=. ./...

# With coverage
go test -cover ./...
```

## 🔧 Configuration

FileOps uses a flexible configuration system. Create a `config.yaml` file:

```yaml
performance:
  max_workers: 0          # Auto-detect
  memory_limit: "80%"
  chunk_size: "64MB"

operations:
  hash_algorithm: "blake2b"
  duplicate_threshold: 0.99
  similarity_threshold: 0.85

ai:
  enabled: true
  model_cache: "./models"
  python_service_url: "http://localhost:8001"

logging:
  level: "info"
  file: "fileops.log"
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Go Team](https://golang.org/) for the excellent language and tools
- [PyTorch Team](https://pytorch.org/) for the ML framework
- All contributors and users of this project

## 🐛 Issues & Support

- 🐞 [Report bugs](https://github.com/a4abhishek/fileops/issues)
- 💡 [Request features](https://github.com/a4abhishek/fileops/issues)
- ❓ [Ask questions](https://github.com/a4abhishek/fileops/discussions)

## 📊 Status

- ✅ Core file operations
- ✅ High-performance deduplication
- ✅ CLI interface
- 🚧 AI-powered features (in progress)
- 🚧 Web UI (in progress)
- 📋 REST API (planned)

---

⭐ **Star this repository if you find it useful!**
