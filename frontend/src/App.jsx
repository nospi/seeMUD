import { useState, useEffect, useRef } from 'react';
import './App.css';
import {
    ConnectToMUD,
    DisconnectFromMUD,
    SendCommand,
    GetOutput,
    GetConnectionStatus
} from "../wailsjs/go/main/App";

function App() {
    const [connected, setConnected] = useState(false);
    const [connecting, setConnecting] = useState(false);
    const [output, setOutput] = useState([]);
    const [inputValue, setInputValue] = useState('');
    const [commandHistory, setCommandHistory] = useState([]);
    const [historyIndex, setHistoryIndex] = useState(-1);

    const outputEndRef = useRef(null);
    const inputRef = useRef(null);

    // Auto-scroll to bottom when new output arrives
    useEffect(() => {
        outputEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }, [output]);

    // Focus input on mount
    useEffect(() => {
        inputRef.current?.focus();
    }, []);

    // Poll for output when connected
    useEffect(() => {
        if (!connected) return;

        const pollOutput = async () => {
            try {
                const lines = await GetOutput();
                if (lines && lines.length > 0) {
                    setOutput(prev => [...prev, ...lines]);
                }
            } catch (err) {
                console.error("Error getting output:", err);
            }
        };

        const interval = setInterval(pollOutput, 100);
        return () => clearInterval(interval);
    }, [connected]);

    const handleConnect = async () => {
        setConnecting(true);
        try {
            await ConnectToMUD("localhost", "4001");
            setConnected(true);
            setOutput(prev => [...prev, "🎮 Connected to WolfMUD!", ""]);
        } catch (err) {
            setOutput(prev => [...prev, `❌ Connection failed: ${err.message || err}`]);
            console.error("Connection error:", err);
        } finally {
            setConnecting(false);
        }
    };

    const handleDisconnect = async () => {
        try {
            await DisconnectFromMUD();
            setConnected(false);
            setOutput(prev => [...prev, "", "👋 Disconnected from MUD"]);
        } catch (err) {
            console.error("Disconnect error:", err);
        }
    };

    const handleSendCommand = async (e) => {
        e.preventDefault();
        if (!connected) return;

        const command = inputValue; // Don't trim here to preserve empty commands

        // Add to output display (show what was actually sent)
        const displayCommand = command || "(enter)";
        setOutput(prev => [...prev, `> ${displayCommand}`]);

        // Add to history (only non-empty commands)
        if (command.trim()) {
            setCommandHistory(prev => [...prev, command.trim()]);
        }
        setHistoryIndex(-1);

        // Send to MUD
        try {
            await SendCommand(command);
        } catch (err) {
            setOutput(prev => [...prev, `❌ Error: ${err}`]);
        }

        setInputValue('');
    };

    const handleKeyDown = (e) => {
        if (e.key === 'ArrowUp') {
            e.preventDefault();
            if (historyIndex < commandHistory.length - 1) {
                const newIndex = historyIndex + 1;
                setHistoryIndex(newIndex);
                setInputValue(commandHistory[commandHistory.length - 1 - newIndex]);
            }
        } else if (e.key === 'ArrowDown') {
            e.preventDefault();
            if (historyIndex > 0) {
                const newIndex = historyIndex - 1;
                setHistoryIndex(newIndex);
                setInputValue(commandHistory[commandHistory.length - 1 - newIndex]);
            } else if (historyIndex === 0) {
                setHistoryIndex(-1);
                setInputValue('');
            }
        }
    };

    // Parse output for basic formatting
    const formatLine = (line) => {
        // Strip all ANSI escape sequences
        let cleaned = line
            // Remove color codes
            .replace(/\x1b\[[0-9;]*m/g, '')
            // Remove cursor positioning and other ANSI sequences
            .replace(/\x1b\[[0-9;]*[a-zA-Z]/g, '')
            // Remove cursor save/restore
            .replace(/\x1b[78]/g, '')
            // Remove screen clear
            .replace(/\x1b\[2J/g, '')
            // Remove any remaining escape characters
            .replace(/\x1b/g, '');

        // Skip empty lines after cleaning
        if (!cleaned.trim()) {
            return '';
        }

        // Detect different content types
        if (cleaned.startsWith('>')) {
            return <span className="command-echo">{cleaned}</span>;
        } else if (cleaned.includes('Exits:')) {
            return <span className="exits">{cleaned}</span>;
        } else if (cleaned.startsWith('🎮') || cleaned.startsWith('👋')) {
            return <span className="system">{cleaned}</span>;
        } else if (cleaned.startsWith('❌')) {
            return <span className="error">{cleaned}</span>;
        }

        return cleaned;
    };

    return (
        <div className="App">
            <div className="header">
                <h1>🎮 See-MUD Visual Client</h1>
                <div className="connection-status">
                    {connected ? (
                        <>
                            <span className="status-connected">● Connected</span>
                            <button onClick={handleDisconnect} className="btn-disconnect">
                                Disconnect
                            </button>
                        </>
                    ) : (
                        <button
                            onClick={handleConnect}
                            disabled={connecting}
                            className="btn-connect"
                        >
                            {connecting ? 'Connecting...' : 'Connect to WolfMUD'}
                        </button>
                    )}
                </div>
            </div>

            <div className="main-content">
                <div className="terminal-container">
                    <div className="terminal-output">
                        {output.map((line, index) => (
                            <div key={index} className="output-line">
                                {formatLine(line)}
                            </div>
                        ))}
                        <div ref={outputEndRef} />
                    </div>

                    <form onSubmit={handleSendCommand} className="terminal-input">
                        <span className="prompt">&gt;</span>
                        <input
                            ref={inputRef}
                            type="text"
                            value={inputValue}
                            onChange={(e) => setInputValue(e.target.value)}
                            onKeyDown={handleKeyDown}
                            disabled={!connected}
                            placeholder={connected ? "Enter command..." : "Connect first"}
                            className="command-input"
                        />
                    </form>
                </div>

                <div className="sidebar">
                    <div className="panel">
                        <h3>🏠 Room View</h3>
                        <div className="image-placeholder">
                            <p>Image generation will appear here</p>
                        </div>
                    </div>

                    <div className="panel">
                        <h3>📦 Items & Mobs</h3>
                        <div className="entity-list">
                            <p>Detected entities will appear here</p>
                        </div>
                    </div>
                </div>
            </div>

            <div className="footer">
                <div className="help-text">
                    Press ↑↓ for command history | Type 'help' for commands | Type 'QUIT' to exit MUD
                </div>
            </div>
        </div>
    );
}

export default App;