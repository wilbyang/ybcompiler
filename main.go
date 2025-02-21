package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

type CompilerOptions struct {
	Compiler string   `json:"compiler"` // "clang", "gcc", "javac"
	Output   string   `json:"output"`   // "llvm-ir", "asm", "bytecode"
	Flags    []string `json:"flags"`    // Additional compiler flags
}

type CompileRequest struct {
	Options CompilerOptions `json:"options"`
}

type CompileResponse struct {
	Success    bool   `json:"success"`
	Output     string `json:"output"`
	Error      string `json:"error,omitempty"`
	SourceCode string `json:"source_code"`
	Language   string `json:"language"` // "c" or "java"
}

// FileWatcher manages file watching and WebSocket connections
type FileWatcher struct {
	watcher   *fsnotify.Watcher
	sourceDir string
	mu        sync.RWMutex
	clients   map[string]map[*websocket.Conn]bool // filename -> connections
}

func NewFileWatcher(sourceDir string) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &FileWatcher{
		watcher:   watcher,
		sourceDir: sourceDir,
		clients:   make(map[string]map[*websocket.Conn]bool),
	}, nil
}

func (fw *FileWatcher) AddClient(filename string, conn *websocket.Conn) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if _, exists := fw.clients[filename]; !exists {
		fw.clients[filename] = make(map[*websocket.Conn]bool)
		// Start watching the file if it's the first client
		fullPath := filepath.Join(fw.sourceDir, filename)
		if err := fw.watcher.Add(fullPath); err != nil {
			log.Printf("Error watching file %s: %v", fullPath, err)
			return
		}
	}
	fw.clients[filename][conn] = true
}

func (fw *FileWatcher) RemoveClient(filename string, conn *websocket.Conn) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if clients, exists := fw.clients[filename]; exists {
		delete(clients, conn)
		if len(clients) == 0 {
			delete(fw.clients, filename)
			fullPath := filepath.Join(fw.sourceDir, filename)
			fw.watcher.Remove(fullPath)
		}
	}
}

func (fw *FileWatcher) compileJava(filename, sourceCode string, options CompilerOptions) (string, error) {
	tmpDir, err := os.MkdirTemp("", "java-compile-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	// Write source to temporary file
	srcFile := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(srcFile, []byte(sourceCode), 0644); err != nil {
		return "", err
	}

	// Compile Java file
	args := append([]string{srcFile}, options.Flags...)
	cmd := exec.Command("javac", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("compilation failed: %s", output)
	}

	// Get class name (remove .java extension)
	className := filename[:len(filename)-5]

	// Use javap with appropriate flags
	javapArgs := []string{"-p"} // Always show private members
	switch options.Output {
	case "bytecode":
		javapArgs = append(javapArgs, "-c") // Show bytecode
	case "verbose":
		javapArgs = append(javapArgs, "-v") // Show verbose output
	}
	javapArgs = append(javapArgs, filepath.Join(tmpDir, className))

	cmd = exec.Command("javap", javapArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get bytecode: %s", output)
	}

	return string(output), nil
}

func (fw *FileWatcher) compileC(filename, sourceCode string, options CompilerOptions) (string, error) {
	tmpDir, err := os.MkdirTemp("", "c-compile-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	// Write source to temporary file
	srcFile := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(srcFile, []byte(sourceCode), 0644); err != nil {
		return "", err
	}

	var cmd *exec.Cmd
	var outputFile string

	switch options.Output {
	case "llvm-ir":
		outputFile = filepath.Join(tmpDir, filename[:len(filename)-2]+".ll")
		args := append([]string{"-S", "-emit-llvm", srcFile, "-o", outputFile}, options.Flags...)
		cmd = exec.Command(options.Compiler, args...)
	case "asm":
		outputFile = filepath.Join(tmpDir, filename[:len(filename)-2]+".s")
		args := append([]string{"-S", srcFile, "-o", outputFile}, options.Flags...)
		cmd = exec.Command(options.Compiler, args...)
	default:
		return "", fmt.Errorf("unsupported output format: %s", options.Output)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("compilation failed: %s", output)
	}

	// Read the generated output
	result, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read output file: %v", err)
	}

	return string(result), nil
}

func (fw *FileWatcher) compileAndNotify(filename string, options CompilerOptions) {
	fullPath := filepath.Join(fw.sourceDir, filename)

	// Read source code
	sourceCode, err := os.ReadFile(fullPath)
	if err != nil {
		fw.notifyClients(filename, "", "Failed to read source file: "+err.Error(), "", "")
		return
	}

	// Determine language based on file extension
	ext := strings.ToLower(filepath.Ext(filename))
	var language string
	var output string

	// Set default options if not specified
	if options.Compiler == "" {
		switch ext {
		case ".c":
			options.Compiler = "clang"
			if options.Output == "" {
				options.Output = "llvm-ir"
			}
		case ".java":
			options.Compiler = "javac"
			if options.Output == "" {
				options.Output = "bytecode"
			}
		}
	}

	switch ext {
	case ".c":
		language = "c"
		output, err = fw.compileC(filename, string(sourceCode), options)
	case ".java":
		language = "java"
		output, err = fw.compileJava(filename, string(sourceCode), options)
	default:
		fw.notifyClients(filename, "", "Unsupported file type: "+ext, string(sourceCode), "")
		return
	}

	if err != nil {
		fw.notifyClients(filename, "", err.Error(), string(sourceCode), language)
		return
	}

	fw.notifyClients(filename, output, "", string(sourceCode), language)
}

func (fw *FileWatcher) notifyClients(filename, output, errMsg, sourceCode, language string) {
	fw.mu.RLock()
	clients := fw.clients[filename]
	fw.mu.RUnlock()

	response := CompileResponse{
		Success:    errMsg == "",
		Output:     output,
		Error:      errMsg,
		SourceCode: sourceCode,
		Language:   language,
	}

	for conn := range clients {
		if err := conn.WriteJSON(response); err != nil {
			log.Printf("Error sending to client: %v", err)
			fw.RemoveClient(filename, conn)
		}
	}
}

func (fw *FileWatcher) Start() {
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				filename := filepath.Base(event.Name)
				fw.compileAndNotify(filename, CompilerOptions{})
			}
		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func handleWebSocket(fw *FileWatcher, w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	if filename == "" {
		http.Error(w, "Missing file parameter", http.StatusBadRequest)
		return
	}

	// Verify file exists
	fullPath := filepath.Join(fw.sourceDir, filename)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	// Add client and set up cleanup
	fw.AddClient(filename, conn)
	defer fw.RemoveClient(filename, conn)

	// Default compiler options
	options := CompilerOptions{}

	// Do initial compilation
	fw.compileAndNotify(filename, options)

	// Handle incoming messages for compiler options updates
	for {
		var req CompileRequest
		if err := conn.ReadJSON(&req); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Update compilation with new options
		fw.compileAndNotify(filename, req.Options)
	}
}

func main() {
	port := flag.String("port", "8080", "port to listen on")
	sourceDir := flag.String("dir", ".", "directory containing source files to watch")
	flag.Parse()

	// Create and start file watcher
	fw, err := NewFileWatcher(*sourceDir)
	if err != nil {
		log.Fatal("Failed to create file watcher:", err)
	}
	defer fw.watcher.Close()

	// Start the file watcher in a goroutine
	go fw.Start()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(fw, w, r)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})

	log.Printf("Server starting on port %s, watching directory: %s", *port, *sourceDir)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
