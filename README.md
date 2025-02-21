# IR Code Watcher

A real-time compiler output visualization tool that helps developers understand how their source code translates to intermediate representations (IR) and assembly. Currently supports:
- C → LLVM IR / Assembly (via Clang or GCC)
- Java → JVM Bytecode (normal or verbose output)

## Value Proposition

This tool is particularly valuable for:

1. **Education**
   - Learn how high-level code translates to IR or assembly
   - Understand compiler optimizations and their effects
   - Study differences between compiler implementations
   - Compare output between different optimization levels

2. **Development**
   - Debug complex performance issues
   - Optimize code by understanding the IR/assembly output
   - Compare different implementations side-by-side
   - Experiment with compiler flags and optimizations

3. **Research**
   - Analyze compiler behavior with different flags
   - Study code transformations across compiler versions
   - Compare different compilation strategies
   - Investigate optimization effects

## Features

- **Real-time Compilation**: Watch files for changes and see IR/assembly updates instantly
- **Configurable Compilation**:
  - Choose between different compilers (Clang/GCC for C)
  - Select output format (LLVM IR/Assembly for C, Bytecode/Verbose for Java)
  - Customize compiler flags (e.g., optimization levels, warnings)
- **Multi-language Support**: 
  - C with LLVM IR or assembly output
  - Java with bytecode or verbose class file information
- **Version Management**:
  - Capture and compare different versions of code
  - Side-by-side diff view for both source and output
  - Timestamp tracking for version history
- **Rich Comparison Tools**:
  - Highlight changes between versions in source code
  - Show differences in IR/assembly output
  - Compare outputs with different compiler settings
- **Modern Web Interface**:
  - Real-time updates via WebSocket
  - Syntax-highlighted output
  - Responsive design for better readability

## Prerequisites

- Go 1.24 or later
- One or more of the following compilers:
  - LLVM/Clang for C (LLVM IR output)
  - GCC for C (assembly output)
  - JDK for Java (bytecode output)
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

# Check C compilers
clang --version
gcc --version

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
   - Enter the filename to watch
   - Select your preferred compiler and output format
   - Add any compiler flags (e.g., -O2 for optimization)
   - Watch real-time updates as you modify the file
   - Capture versions to compare different implementations or compiler settings

## Command Line Options

- `-port`: HTTP server port (default: 8080)
- `-dir`: Directory to watch for source files (default: current directory)

## Architecture

The system consists of three main components:

1. **File Watcher**: Monitors source files for changes using fsnotify
2. **Compiler Bridge**: 
   - Manages multiple compiler configurations
   - Handles different output formats
   - Processes compiler flags
3. **WebSocket Server**: Provides real-time updates to the web interface

## Development

To extend the system with new features:

1. Add new compiler support:
   - Implement compilation function in `main.go`
   - Update language detection in `compileAndNotify`
   - Add compiler options in frontend

2. Add new output formats:
   - Update CompilerOptions struct
   - Implement format-specific compilation logic
   - Add UI support in frontend

## Limitations

- Only supports single-file compilation
- Requires local installation of compilers
- Limited to specific output formats
- No support for complex build configurations
- Memory limited by browser for large diffs

## Contributing

Contributions are welcome! Some areas for improvement:

- Additional compiler support
- More output formats
- Build system integration
- Syntax highlighting for different outputs
- Persistent version history
- Multi-file project support
- Compiler optimization visualization
- Performance analysis tools

## License

MIT License - See LICENSE file for details

## Acknowledgments

- LLVM/Clang project
- GCC project
- OpenJDK tools
- fsnotify and gorilla/websocket Go packages 