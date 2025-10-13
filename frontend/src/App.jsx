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
    RegenerateRoomImage,
    RegenerateRoomImageWithPrompt,
    GetCurrentRoom,
    GetRoomImage,
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
    const [sidebarWidth, setSidebarWidth] = useState(600); // Default to 600px
    const [isResizing, setIsResizing] = useState(false);
    const [showPromptInput, setShowPromptInput] = useState(false);
    const [customPrompt, setCustomPrompt] = useState('');

    const outputEndRef = useRef(null);
    const inputRef = useRef(null);
    const generatingRef = useRef(false);

    // Auto-scroll to bottom when new output arrives
    useEffect(() => {
        outputEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }, [output]);

    // Focus input on mount
    useEffect(() => {
        inputRef.current?.focus();
    }, []);

    // Handle mouse move for resizing
    useEffect(() => {
        const handleMouseMove = (e) => {
            if (!isResizing) return;

            const newWidth = window.innerWidth - e.clientX;
            // Constrain between 300px and 80% of window width
            const minWidth = 300;
            const maxWidth = window.innerWidth * 0.8;
            setSidebarWidth(Math.min(Math.max(newWidth, minWidth), maxWidth));
        };

        const handleMouseUp = () => {
            setIsResizing(false);
        };

        if (isResizing) {
            document.addEventListener('mousemove', handleMouseMove);
            document.addEventListener('mouseup', handleMouseUp);
            document.body.style.cursor = 'col-resize';
            document.body.style.userSelect = 'none';
        }

        return () => {
            document.removeEventListener('mousemove', handleMouseMove);
            document.removeEventListener('mouseup', handleMouseUp);
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
        };
    }, [isResizing]);

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
                    // Check if room has changed
                    setCurrentRoom(prevRoom => {
                        if (prevRoom.name !== room.name) {
                            // Room changed, check for cached image
                            checkCachedImage(room);
                        }
                        return room;
                    });
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
                setSdAvailable(prev => {
                    // If SD just became available and we have a room without image, generate
                    if (!prev && available && currentRoom.name && !roomImage && !generatingRef.current) {
                        console.log("SD just became available, checking for image generation needs");
                        setTimeout(() => {
                            checkCachedImage();
                        }, 500);
                    }
                    return available;
                });
            } catch (err) {
                setSdAvailable(false);
            }
        };

        checkSD();
        const interval = setInterval(checkSD, 10000); // Check every 10 seconds
        return () => clearInterval(interval);
    }, [currentRoom.name, roomImage]);

    const handleConnect = async () => {
        setConnecting(true);
        try {
            await ConnectToMUD("localhost", "4001");
            setConnected(true);
            setOutput(prev => [...prev, "🎮 Connected to WolfMUD!", ""]);
            // Check for cached image after initial connection with longer delay
            // to ensure room data is loaded
            setTimeout(() => {
                console.log("Initial connection - checking for cached image");
                checkCachedImage();
            }, 1500);
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

    // Check if there's a cached image for the current room
    const checkCachedImage = async (roomData = null) => {
        try {
            const cachedImage = await GetRoomImage();
            if (cachedImage) {
                console.log("Found cached image for room");
                setRoomImage(`data:image/png;base64,${cachedImage}`);
                return true; // Image found
            } else {
                // Use passed roomData or fetch current room
                const room = roomData || await GetCurrentRoom();
                console.log("No cached image found for room:", room);
                console.log("SD Available:", sdAvailable, "Generating:", generatingRef.current);
                setRoomImage(null);

                // Auto-generate image for new rooms if SD is available
                // We need to ensure room has a name and SD is ready
                if (room && room.name && room.name.trim() !== "") {
                    console.log("Room is valid, checking SD and generation status...");
                    // Try auto-generation after a delay to ensure SD status is updated
                    setTimeout(async () => {
                        const sdStatus = await CheckSDStatus();
                        console.log("SD Status check:", sdStatus, "Generating:", generatingRef.current);
                        if (sdStatus && !generatingRef.current) {
                            console.log("Auto-generating image for room:", room.name);
                            // For auto-generation, always generate new (not regenerate)
                            autoGenerateImage(room);
                        }
                    }, 1000); // Slightly longer delay to ensure everything is ready
                }
                return false; // No image found
            }
        } catch (err) {
            console.error("Error checking cached image:", err);
            return false;
        }
    };

    // Auto-generate function that always generates new (for rooms without images)
    const autoGenerateImage = async (roomData = null) => {
        const room = roomData || currentRoom;
        console.log("autoGenerateImage called with room:", room, "generating:", generatingRef.current);

        if (!room || !room.name || generatingRef.current) {
            console.log("Aborting auto-generation - room:", !!room, "name:", room?.name, "generating:", generatingRef.current);
            return;
        }

        console.log("Starting auto-generation for room:", room.name);
        generatingRef.current = true;
        setGeneratingImage(true);
        try {
            console.log("Calling RegenerateRoomImage for room:", room.name);
            // Use RegenerateRoomImage to bypass cache and always generate fresh
            const imageBase64 = await RegenerateRoomImage();
            console.log("Got image, setting room image");
            setRoomImage(`data:image/png;base64,${imageBase64}`);
        } catch (err) {
            console.error("Auto image generation failed:", err);
            setOutput(prev => [...prev, `❌ Auto image generation failed: ${err.message || err}`]);
        } finally {
            console.log("Auto-generation completed, resetting flags");
            generatingRef.current = false;
            setGeneratingImage(false);
        }
    };

    const handleGenerateImage = async () => {
        if (!sdAvailable || !currentRoom.name || generatingRef.current) return;

        generatingRef.current = true;
        setGeneratingImage(true);
        try {
            // Use RegenerateRoomImage if we already have an image, otherwise use GenerateRoomImage
            const imageBase64 = roomImage
                ? await RegenerateRoomImage()
                : await GenerateRoomImage();
            setRoomImage(`data:image/png;base64,${imageBase64}`);
        } catch (err) {
            console.error("Image generation failed:", err);
            setOutput(prev => [...prev, `❌ Image generation failed: ${err.message || err}`]);
        } finally {
            generatingRef.current = false;
            setGeneratingImage(false);
        }
    };

    const handleGenerateWithCustomPrompt = async () => {
        if (!sdAvailable || !currentRoom.name || generatingRef.current || !customPrompt.trim()) return;

        generatingRef.current = true;
        setGeneratingImage(true);
        try {
            const imageBase64 = await RegenerateRoomImageWithPrompt(customPrompt.trim());
            setRoomImage(`data:image/png;base64,${imageBase64}`);
        } catch (err) {
            console.error("Custom image generation failed:", err);
            setOutput(prev => [...prev, `❌ Custom image generation failed: ${err.message || err}`]);
        } finally {
            generatingRef.current = false;
            setGeneratingImage(false);
        }
    };

    const togglePromptInput = () => {
        setShowPromptInput(prev => !prev);
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
                <h1>🎮 SeeMUD Visual Client</h1>
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

                <div
                    className="resize-handle"
                    onMouseDown={() => setIsResizing(true)}
                />

                <div className="sidebar" style={{ width: `${sidebarWidth}px` }}>
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
                                        <p>{currentRoom.name ? 'No image yet - click Generate to create one' : 'Explore a room to see images'}</p>
                                    </div>
                                )}
                            </div>
                            <div className="image-controls">
                                {roomImage ? (
                                    // Split button for regeneration
                                    <>
                                        <div className="btn-split-group">
                                            <button
                                                onClick={handleGenerateImage}
                                                disabled={!sdAvailable || !currentRoom.name || generatingImage}
                                                className="btn-split-main"
                                            >
                                                {generatingImage ? '🎨 Generating...' : '🔄 Regenerate'}
                                            </button>
                                            <button
                                                onClick={togglePromptInput}
                                                disabled={!sdAvailable || !currentRoom.name || generatingImage}
                                                className="btn-split-dropdown"
                                                title="Custom prompt options"
                                            >
                                                ▼
                                            </button>
                                        </div>

                                        {showPromptInput && (
                                            <div className="custom-prompt-section">
                                                <label className="custom-prompt-label">
                                                    Custom instructions:
                                                </label>
                                                <textarea
                                                    className="custom-prompt-textarea"
                                                    value={customPrompt}
                                                    onChange={(e) => setCustomPrompt(e.target.value)}
                                                    disabled={generatingImage}
                                                    placeholder="e.g., darker, more fog, torchlight..."
                                                    rows={3}
                                                />
                                                <button
                                                    onClick={handleGenerateWithCustomPrompt}
                                                    disabled={!sdAvailable || !currentRoom.name || generatingImage || !customPrompt.trim()}
                                                    className="btn-generate-custom"
                                                >
                                                    {generatingImage ? '🎨 Generating...' : '✨ Generate with Custom Prompt'}
                                                </button>
                                            </div>
                                        )}
                                    </>
                                ) : (
                                    // Simple generate button when no image
                                    <button
                                        onClick={handleGenerateImage}
                                        disabled={!sdAvailable || !currentRoom.name || generatingImage}
                                        className="btn-generate"
                                    >
                                        {generatingImage ? '🎨 Generating...' : '🎨 Generate Image'}
                                    </button>
                                )}
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