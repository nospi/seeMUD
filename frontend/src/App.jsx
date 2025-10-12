import { useState, useEffect, useRef } from 'react';
import './App.css';
import { AnsiText } from './ansi.jsx';
import {
    ConnectToMUD,
    DisconnectFromMUD,
    SendCommand,
    GetOutput,
    GetConnectionStatus,
    GenerateRoomImage,
    GetCurrentRoom,
    CheckSDStatus
} from "../wailsjs/go/main/App";

function App() {
    const [connected, setConnected] = useState(false);
    const [connecting, setConnecting] = useState(false);
    const [output, setOutput] = useState([]);
    const [inputValue, setInputValue] = useState('');
    const [commandHistory, setCommandHistory] = useState([]);
    const [historyIndex, setHistoryIndex] = useState(-1);
    const [currentRoom, setCurrentRoom] = useState({});
    const [roomImage, setRoomImage] = useState(null);
    const [generatingImage, setGeneratingImage] = useState(false);
    const [sdAvailable, setSdAvailable] = useState(false);

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

                // Check for room updates
                const room = await GetCurrentRoom();
                if (room && (room.name || room.description)) {
                    setCurrentRoom(room);
                }
            } catch (err) {
                console.error("Error getting output:", err);
            }
        };

        const interval = setInterval(pollOutput, 100);
        return () => clearInterval(interval);
    }, [connected]);

    // Check SD status periodically
    useEffect(() => {
        const checkSD = async () => {
            try {
                const available = await CheckSDStatus();
                setSdAvailable(available);
            } catch (err) {
                setSdAvailable(false);
            }
        };

        checkSD();
        const interval = setInterval(checkSD, 10000); // Check every 10 seconds
        return () => clearInterval(interval);
    }, []);

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

    const handleGenerateImage = async () => {
        if (!sdAvailable || !currentRoom.name || generatingImage) return;

        setGeneratingImage(true);
        try {
            const imageBase64 = await GenerateRoomImage();
            setRoomImage(`data:image/png;base64,${imageBase64}`);
        } catch (err) {
            console.error("Image generation failed:", err);
            setOutput(prev => [...prev, `❌ Image generation failed: ${err.message || err}`]);
        } finally {
            setGeneratingImage(false);
        }
    };

    // Parse output for basic formatting with ANSI support
    const formatLine = (line) => {
        // Skip completely empty lines
        if (!line.trim()) {
            return '';
        }

        // Get clean text for content type detection
        const cleaned = line
            .replace(/\x1b\[[0-9;]*m/g, '')
            .replace(/\x1b\[[0-9;]*[a-zA-Z]/g, '')
            .replace(/\x1b[78]/g, '')
            .replace(/\x1b\[2J/g, '')
            .replace(/\x1b/g, '');

        // Detect different content types and wrap with appropriate classes
        if (cleaned.startsWith('>')) {
            return <span className="command-echo"><AnsiText>{line}</AnsiText></span>;
        } else if (cleaned.includes('Exits:')) {
            return <span className="exits"><AnsiText>{line}</AnsiText></span>;
        } else if (cleaned.startsWith('🎮') || cleaned.startsWith('👋')) {
            return <span className="system"><AnsiText>{line}</AnsiText></span>;
        } else if (cleaned.startsWith('❌')) {
            return <span className="error"><AnsiText>{line}</AnsiText></span>;
        }

        // Default: render with ANSI support
        return <AnsiText>{line}</AnsiText>;
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
                        <div className="room-info">
                            {currentRoom.name && (
                                <div className="room-name">
                                    <strong>{currentRoom.name}</strong>
                                </div>
                            )}
                            <div className="image-container">
                                {roomImage ? (
                                    <img
                                        src={roomImage}
                                        alt="Generated room view"
                                        className="room-image"
                                    />
                                ) : (
                                    <div className="image-placeholder">
                                        <p>{currentRoom.name ? 'Ready to generate image' : 'Explore a room to generate images'}</p>
                                    </div>
                                )}
                            </div>
                            <div className="image-controls">
                                <button
                                    onClick={handleGenerateImage}
                                    disabled={!sdAvailable || !currentRoom.name || generatingImage}
                                    className="btn-generate"
                                >
                                    {generatingImage ? '🎨 Generating...' : '🎨 Generate Image'}
                                </button>
                                <div className="sd-status">
                                    SD: <span className={sdAvailable ? 'status-ok' : 'status-error'}>
                                        {sdAvailable ? '✅ Ready' : '❌ Not Available'}
                                    </span>
                                </div>
                            </div>
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