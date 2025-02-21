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

func (fw *FileWatcher) compileJava(filename, sourceCode string) (string, error) {
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
	cmd := exec.Command("javac", srcFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("compilation failed: %s", output)
	}

	// Get class name (remove .java extension)
	className := filename[:len(filename)-5]

	// Use javap to get bytecode
	cmd = exec.Command("javap", "-c", "-p", "-v", filepath.Join(tmpDir, className))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get bytecode: %s", output)
	}

	return string(output), nil
}

func (fw *FileWatcher) compileC(filename, sourceCode string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "clang-compile-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	// Write source to temporary file
	srcFile := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(srcFile, []byte(sourceCode), 0644); err != nil {
		return "", err
	}

	// Generate output file path
	llFileName := filename[:len(filename)-2] + ".ll" // Remove .c and add .ll
	llvmFile := filepath.Join(tmpDir, llFileName)

	// Run clang command
	cmd := exec.Command("clang", "-S", "-emit-llvm", srcFile, "-o", llvmFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("compilation failed: %s", output)
	}

	// Read the generated LLVM IR
	llvmIR, err := os.ReadFile(llvmFile)
	if err != nil {
		return "", fmt.Errorf("failed to read LLVM IR: %v", err)
	}

	return string(llvmIR), nil
}

func (fw *FileWatcher) compileAndNotify(filename string) {
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

	switch ext {
	case ".c":
		language = "c"
		output, err = fw.compileC(filename, string(sourceCode))
	case ".java":
		language = "java"
		output, err = fw.compileJava(filename, string(sourceCode))
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
				fw.compileAndNotify(filename)
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

	// Do initial compilation
	fw.compileAndNotify(filename)

	// Keep connection alive until client disconnects
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
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
