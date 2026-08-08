// web/main.go
package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Config struct {
	Mode       string `json:"mode"`
	ListenPort string `json:"listen_port"`
	Backend    string `json:"backend,omitempty"`
	ServerAddr string `json:"server_addr,omitempty"`
}

type Status struct {
	Running     bool   `json:"running"`
	Mode        string `json:"mode"`
	ListenPort  string `json:"listen_port"`
	Backend     string `json:"backend,omitempty"`
	ServerAddr  string `json:"server_addr,omitempty"`
	PID         int    `json:"pid"`
	BytesIn     uint64 `json:"bytes_in"`
	BytesOut    uint64 `json:"bytes_out"`
	Connections int    `json:"connections"`
	StartTime   string `json:"start_time"`
}

var (
	enzoCmd   *exec.Cmd
	enzoMutex sync.Mutex
	currentConfig Config
	currentStatus Status
	statusMutex   sync.RWMutex
)

var indexTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ENZO Configuration</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #1a1a2e;
            color: #eee;
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
            padding: 20px;
        }
        .container {
            background: #16213e;
            border-radius: 12px;
            padding: 30px;
            width: 100%;
            max-width: 500px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.3);
        }
        h1 {
            text-align: center;
            margin-bottom: 30px;
            color: #e94560;
        }
        .mode-toggle {
            display: flex;
            gap: 10px;
            margin-bottom: 25px;
        }
        .mode-toggle label {
            flex: 1;
            text-align: center;
            padding: 12px;
            background: #0f3460;
            border-radius: 8px;
            cursor: pointer;
            transition: all 0.3s;
            border: 2px solid transparent;
        }
        .mode-toggle input {
            display: none;
        }
        .mode-toggle input:checked + label {
            background: #e94560;
            border-color: #e94560;
        }
        .config-section {
            background: #0f3460;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 20px;
        }
        .config-section h3 {
            margin-bottom: 15px;
            color: #e94560;
            font-size: 14px;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        .form-group {
            margin-bottom: 15px;
        }
        .form-group label {
            display: block;
            margin-bottom: 5px;
            color: #aaa;
            font-size: 14px;
        }
        .form-group input {
            width: 100%;
            padding: 10px 15px;
            border: 1px solid #333;
            border-radius: 6px;
            background: #1a1a2e;
            color: #fff;
            font-size: 16px;
        }
        .form-group input:focus {
            outline: none;
            border-color: #e94560;
        }
        .status-section {
            background: #0f3460;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 20px;
        }
        .status-row {
            display: flex;
            justify-content: space-between;
            padding: 8px 0;
            border-bottom: 1px solid #333;
        }
        .status-row:last-child {
            border-bottom: none;
        }
        .status-indicator {
            display: inline-block;
            width: 12px;
            height: 12px;
            border-radius: 50%;
            margin-right: 8px;
        }
        .status-indicator.running {
            background: #4ade80;
            box-shadow: 0 0 10px #4ade80;
        }
        .status-indicator.stopped {
            background: #ef4444;
        }
        .btn-group {
            display: flex;
            gap: 10px;
        }
        .btn {
            flex: 1;
            padding: 12px 20px;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            cursor: pointer;
            transition: all 0.3s;
        }
        .btn-start {
            background: #4ade80;
            color: #000;
        }
        .btn-start:hover {
            background: #22c55e;
        }
        .btn-stop {
            background: #ef4444;
            color: #fff;
        }
        .btn-stop:hover {
            background: #dc2626;
        }
        .btn:disabled {
            opacity: 0.5;
            cursor: not-allowed;
        }
        .hidden {
            display: none;
        }
        .error {
            color: #ef4444;
            text-align: center;
            margin-top: 10px;
        }
        .info {
            color: #aaa;
            font-size: 12px;
            margin-top: 4px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>⚡ ENZO Configuration</h1>

        <div class="mode-toggle">
            <input type="radio" id="mode-server" name="mode" value="server" onchange="updateUI()">
            <label for="mode-server">Server Mode</label>
            <input type="radio" id="mode-client" name="mode" value="client" onchange="updateUI()">
            <label for="mode-client">Client Mode</label>
        </div>

        <div id="server-config" class="config-section hidden">
            <h3>Server Configuration</h3>
            <div class="form-group">
                <label>Listen Port</label>
                <input type="text" id="server-listen-port" value="8086">
                <div class="info">Port for ENZO server to listen on</div>
            </div>
            <div class="form-group">
                <label>Backend (InfluxDB)</label>
                <input type="text" id="server-backend" value="influxdb:8086">
                <div class="info">Backend service address</div>
            </div>
        </div>

        <div id="client-config" class="config-section hidden">
            <h3>Client Configuration</h3>
            <div class="form-group">
                <label>Listen Port</label>
                <input type="text" id="client-listen-port" value="8088">
                <div class="info">Port for edge devices to connect to</div>
            </div>
            <div class="form-group">
                <label>Server ENZO Address</label>
                <input type="text" id="client-server-addr" value="192.168.1.50:8086">
                <div class="info">Address of server-side ENZO</div>
            </div>
        </div>

        <div class="status-section">
            <h3>Status</h3>
            <div class="status-row">
                <span><span id="status-indicator" class="status-indicator stopped"></span>Status</span>
                <span id="status-text">Stopped</span>
            </div>
            <div class="status-row">
                <span>Mode</span>
                <span id="status-mode">-</span>
            </div>
            <div class="status-row">
                <span>PID</span>
                <span id="status-pid">-</span>
            </div>
            <div class="status-row">
                <span>Connections</span>
                <span id="status-connections">0</span>
            </div>
            <div class="status-row">
                <span>Bytes In</span>
                <span id="status-bytes-in">0 B</span>
            </div>
            <div class="status-row">
                <span>Bytes Out</span>
                <span id="status-bytes-out">0 B</span>
            </div>
            <div class="status-row">
                <span>Started</span>
                <span id="status-start-time">-</span>
            </div>
        </div>

        <div id="error-msg" class="error hidden"></div>

        <div class="btn-group">
            <button id="btn-start" class="btn btn-start" onclick="startEnzo()">Start</button>
            <button id="btn-stop" class="btn btn-stop" onclick="stopEnzo()" disabled>Stop</button>
        </div>
    </div>

    <script>
        let statusInterval = null;

        function updateUI() {
            const mode = document.querySelector('input[name="mode"]:checked')?.value || 'server';
            document.getElementById('server-config').classList.toggle('hidden', mode !== 'server');
            document.getElementById('client-config').classList.toggle('hidden', mode !== 'client');
        }

        async function startEnzo() {
            const mode = document.querySelector('input[name="mode"]:checked')?.value || 'server';
            let config = { mode };

            if (mode === 'server') {
                config.listen_port = document.getElementById('server-listen-port').value;
                config.backend = document.getElementById('server-backend').value;
            } else {
                config.listen_port = document.getElementById('client-listen-port').value;
                config.server_addr = document.getElementById('client-server-addr').value;
            }

            const resp = await fetch('/api/start', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(config)
            });

            const data = await resp.json();
            if (data.error) {
                showError(data.error);
            } else {
                hideError();
                updateStatus(data.status);
                document.getElementById('btn-start').disabled = true;
                document.getElementById('btn-stop').disabled = false;
                startPolling();
            }
        }

        async function stopEnzo() {
            const resp = await fetch('/api/stop', { method: 'POST' });
            const data = await resp.json();
            if (data.error) {
                showError(data.error);
            } else {
                hideError();
                updateStatus(data.status);
                document.getElementById('btn-start').disabled = false;
                document.getElementById('btn-stop').disabled = true;
                stopPolling();
            }
        }

        function updateStatus(status) {
            const indicator = document.getElementById('status-indicator');
            indicator.className = 'status-indicator ' + (status.running ? 'running' : 'stopped');
            document.getElementById('status-text').textContent = status.running ? 'Running' : 'Stopped';
            document.getElementById('status-mode').textContent = status.mode || '-';
            document.getElementById('status-pid').textContent = status.pid || '-';
            document.getElementById('status-connections').textContent = status.connections || 0;
            document.getElementById('status-bytes-in').textContent = formatBytes(status.bytes_in);
            document.getElementById('status-bytes-out').textContent = formatBytes(status.bytes_out);
            document.getElementById('status-start-time').textContent = status.start_time || '-';
        }

        function formatBytes(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        function startPolling() {
            if (statusInterval) return;
            statusInterval = setInterval(async () => {
                const resp = await fetch('/api/status');
                const data = await resp.json();
                if (data.status) {
                    updateStatus(data.status);
                }
            }, 2000);
        }

        function stopPolling() {
            if (statusInterval) {
                clearInterval(statusInterval);
                statusInterval = null;
            }
        }

        function showError(msg) {
            const el = document.getElementById('error-msg');
            el.textContent = msg;
            el.classList.remove('hidden');
        }

        function hideError() {
            document.getElementById('error-msg').classList.add('hidden');
        }

        // Initialize
        document.getElementById('mode-server').checked = true;
        updateUI();

        // Fetch initial status
        fetch('/api/status').then(r => r.json()).then(data => {
            if (data.status) {
                updateStatus(data.status);
                if (data.status.running) {
                    document.getElementById('btn-start').disabled = true;
                    document.getElementById('btn-stop').disabled = false;
                    startPolling();
                }
            }
        });
    </script>
</body>
</html>
`

func main() {
	// Serve static HTML
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.New("index").Parse(indexTemplate)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		tmpl.Execute(w, nil)
	})

	// API endpoints
	http.HandleFunc("/api/start", handleStart)
	http.HandleFunc("/api/stop", handleStop)
	http.HandleFunc("/api/status", handleStatus)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8282"
	}

	log.Printf("ENZO Web UI starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var config Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeJSON(w, map[string]interface{}{"error": err.Error()})
		return
	}

	enzoMutex.Lock()
	defer enzoMutex.Unlock()

	// Stop existing instance
	if enzoCmd != nil && enzoCmd.Process != nil {
		enzoCmd.Process.Kill()
		enzoCmd.Wait()
	}

	// Build ENZO command based on mode
	args := []string{}

	if config.Mode == "server" {
		// Server mode: listen on port, forward to backend
		args = []string{"-listen", ":" + config.ListenPort, "-dest", config.Backend}
	} else {
		// Client mode: listen on port, forward to server ENZO
		args = []string{"-listen", ":" + config.ListenPort, "-dest", config.ServerAddr}
	}

	// Find enzo binary
	enzoBin := findEnzoBinary()

	cmd := exec.Command(enzoBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		writeJSON(w, map[string]interface{}{"error": err.Error()})
		return
	}

	currentConfig = config
	statusMutex.Lock()
	currentStatus = Status{
		Running:    true,
		Mode:       config.Mode,
		ListenPort: config.ListenPort,
		PID:        cmd.Process.Pid,
		StartTime:  time.Now().Format("15:04:05"),
	}
	if config.Mode == "server" {
		currentStatus.Backend = config.Backend
	} else {
		currentStatus.ServerAddr = config.ServerAddr
	}
	statusMutex.Unlock()

	enzoCmd = cmd

	// Monitor process
	go func() {
		cmd.Wait()
		statusMutex.Lock()
		currentStatus.Running = false
		statusMutex.Unlock()
	}()

	writeJSON(w, map[string]interface{}{"status": getStatus()})
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	enzoMutex.Lock()
	defer enzoMutex.Unlock()

	if enzoCmd != nil && enzoCmd.Process != nil {
		enzoCmd.Process.Kill()
		enzoCmd.Wait()
		enzoCmd = nil
	}

	statusMutex.Lock()
	currentStatus.Running = false
	statusMutex.Unlock()

	writeJSON(w, map[string]interface{}{"status": getStatus()})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{"status": getStatus()})
}

func getStatus() Status {
	statusMutex.RLock()
	defer statusMutex.RUnlock()
	return currentStatus
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func findEnzoBinary() string {
	// Check common locations
	paths := []string{
		"/app/enzo",
		"/usr/local/bin/enzo",
		"./enzo",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Fallback to current directory
	return "/app/enzo"
}
