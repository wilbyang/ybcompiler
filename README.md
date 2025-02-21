# IR Code Watcher

A real-time compiler output visualization tool that helps developers understand how their source code translates to intermediate representations (IR). Currently supports:
- C → LLVM IR
- Java → JVM Bytecode

## Value Proposition

This tool is particularly valuable for:

1. **Education**
   - Learn how high-level code translates to IR
   - Understand compiler optimizations
   - Study differences between language implementations

2. **Development**
   - Debug complex performance issues
   - Optimize code by understanding the IR
   - Compare different implementations side-by-side

3. **Research**
   - Analyze compiler behavior
   - Study code transformations
   - Compare different versions of code and their IR

## Features

- **Real-time Compilation**: Watch files for changes and see IR updates instantly
- **Multi-language Support**: C and Java support with extensible architecture
- **Side-by-side Comparison**: Compare different versions of code and their IR
- **Diff View**: Highlight changes between versions in both source and IR
- **Web Interface**: Easy-to-use browser-based interface
- **Syntax Highlighting**: Clear visualization of code and IR

## Prerequisites

- Go 1.24 or later
- LLVM/Clang for C compilation
- JDK (Java Development Kit) for Java compilation
- Modern web browser

## Installation

1. Clone the repository:
```bash
git clone https://github.com/wilbyang/ybcompiler.git
cd ybcompiler
```

2. Install dependencies:
```bash
go mod download
```

3. Verify your environment:
```bash
# Check Go version
go version

# Check Clang installation
clang --version

# Check Java installation
javac -version
javap -version
```

## Usage

1. Start the server:
```bash
go run main.go -port 8080 -dir ./src
```

2. Open your browser and navigate to:
```
http://localhost:8080
```

3. Create or modify source files in the watched directory:
```bash
# Example C file
echo 'int main() { return 42; }' > src/example.c

# Example Java file
echo 'public class Example { public static void main(String[] args) { System.out.println(42); } }' > src/Example.java
```

4. In the web interface:
   - Enter the filename to watch (e.g., "example.c" or "Example.java")
   - Click "Watch File"
   - Make changes to your source file and see IR updates in real-time
   - Use "Capture Current Version" to save snapshots
   - Compare different versions using the comparison view

## Command Line Options

- `-port`: HTTP server port (default: 8080)
- `-dir`: Directory to watch for source files (default: current directory)

## Architecture

The system consists of three main components:

1. **File Watcher**: Monitors source files for changes using fsnotify
2. **Compiler Bridge**: Interfaces with different compilers (Clang, javac)
3. **WebSocket Server**: Provides real-time updates to the web interface

## Development

To extend the system with new language support:

1. Add a new compilation function in `main.go`
2. Update the language detection in `compileAndNotify`
3. Add appropriate frontend display logic in `index.html`

## Limitations

- Only supports single-file compilation
- Requires local installation of compilers
- Limited to LLVM IR and JVM bytecode output formats
- No support for complex build configurations

## Contributing

Contributions are welcome! Some areas for improvement:

- Additional language support
- More IR output formats
- Build system integration
- Syntax highlighting for IR code
- Configuration options for compiler flags
- Support for multi-file projects

## License

MIT License - See LICENSE file for details

## Acknowledgments

- LLVM/Clang project
- OpenJDK tools
- fsnotify and gorilla/websocket Go packages 