# 🕵️ Gitrob

<p align="center">
  <img src="https://github.com/michenriksen/gitrob/raw/master/static/images/gopher_full.png" alt="Gitrob" width="200" />
</p>

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.8-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

</div>

> Gitrob is a tool to help find potentially sensitive files pushed to public repositories on Github.

## 📋 Table of Contents
- [Features](#-features)
- [Installation](#-installation)
- [Quick Start](#-quick-start)
- [Configuration](#-configuration)
- [Usage](#-usage)
- [Building from Source](#-building-from-source)
- [Contributing](#-contributing)

## ✨ Features
- 🔍 Scans repositories for sensitive files
- 🌐 Web interface for easy analysis
- 🔄 Configurable commit depth scanning
- 👥 Organization member scanning
- 💾 Session saving and loading
- ⚙️ Customizable signature patterns
- 🚀 Multi-threaded processing

## 📥 Installation

### Pre-built Binaries
Download the latest [pre-built release](https://github.com/michenriksen/gitrob/releases) for your platform.

### Using Go
```bash
go get github.com/michenriksen/gitrob
```

## 🚀 Quick Start

1. **Set up GitHub Token**
```bash
export GITROB_ACCESS_TOKEN=your_github_token
```

2. **Run Gitrob**
```bash
gitrob target_organization
```

3. **Access Web Interface**
```
http://localhost:9393
```

## ⚙️ Configuration

### GitHub Access Token
1. [Create a personal access token](https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/)
2. Set it in your environment:
```bash
export GITROB_ACCESS_TOKEN=your_token_here
```

### Signature Configuration
Place your `config.yaml` in one of these locations:
- `./config.yaml`
- `./core/config.yaml`
- `/etc/gitrob/config.yaml`
- `$HOME/.gitrob/config.yaml`

#### Custom Signature Format
```yaml
- name: "sensitive_file"
  type: "content|extension|filename"
  pattern: "regex_pattern"
  description: "What this detects"
  comment: "Additional context"
```

## 🛠️ Usage

### Command Format
```bash
gitrob [options] target [target2] ... [targetN]
```

### Options
| Option | Description | Default |
|--------|-------------|---------|
| -bind-address | Web server bind address | 127.0.0.1 |
| -commit-depth | Number of commits to process | 500 |
| -debug | Enable debug output | false |
| -github-access-token | GitHub API token | - |
| -load | Load session file | - |
| -no-expand-orgs | Don't scan org members | false |
| -port | Web server port | 9393 |
| -save | Save session to file | - |
| -silent | Suppress output | false |
| -threads | Concurrent threads | CPU cores |

### Session Management

#### Save Session
```bash
gitrob -save ~/gitrob-session.json acmecorp
```

#### Load Session
```bash
gitrob -load ~/gitrob-session.json
```

## 🔨 Building from Source

### Prerequisites
- Go >= 1.8
- Git

### Build Steps
1. **Clone Repository**
```bash
git clone https://github.com/michenriksen/gitrob.git
cd gitrob
```

2. **Build**
```bash
chmod +x build.sh
./build.sh
```

This creates binaries in the `build` directory for:
- Linux (amd64)
- macOS (amd64)
- Windows (amd64)

For single platform build:
```bash
go build
```

## 🤝 Contributing
Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

## 📄 License
This project is licensed under the MIT License - see the LICENSE file for details.
