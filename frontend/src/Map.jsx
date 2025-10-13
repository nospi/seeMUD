import { useState, useEffect, useRef } from 'react';
import './Map.css';
import { GetMapData } from "../wailsjs/go/main/App";

const CELL_SIZE = 40; // Size of each room cell in pixels
const GRID_PADDING = 20; // Padding around the map

function Map({ connected }) {
    const [mapData, setMapData] = useState(null);
    const [selectedRoom, setSelectedRoom] = useState(null);
    const [zLevel, setZLevel] = useState(0);
    const canvasRef = useRef(null);

    // Poll for map data when connected
    useEffect(() => {
        if (!connected) return;

        const pollMap = async () => {
            try {
                const data = await GetMapData();
                setMapData(data);

                // Auto-adjust Z level to current room
                if (data.current_room_id) {
                    const currentRoom = data.rooms.find(r => r.id === data.current_room_id);
                    if (currentRoom) {
                        setZLevel(currentRoom.z);
                    }
                }
            } catch (err) {
                console.error("Error getting map data:", err);
            }
        };

        // Initial fetch
        pollMap();

        // Poll every 500ms
        const interval = setInterval(pollMap, 500);
        return () => clearInterval(interval);
    }, [connected]);

    // Render map to canvas
    useEffect(() => {
        if (!mapData || !canvasRef.current) return;

        const canvas = canvasRef.current;
        const ctx = canvas.getContext('2d');

        // Clear canvas
        ctx.fillStyle = '#0f0f23';
        ctx.fillRect(0, 0, canvas.width, canvas.height);

        // Filter rooms by current Z level
        const roomsAtLevel = mapData.rooms.filter(r => r.z === zLevel);

        if (roomsAtLevel.length === 0) {
            ctx.fillStyle = '#666';
            ctx.font = '14px Courier New';
            ctx.fillText('No rooms at this level', 10, 30);
            return;
        }

        // Calculate offsets to centre the map
        const bounds = mapData.bounds;
        const offsetX = GRID_PADDING - (bounds.min_x * CELL_SIZE);
        const offsetY = GRID_PADDING - (bounds.min_y * CELL_SIZE);

        // Draw grid lines (optional)
        ctx.strokeStyle = '#1a1a2e';
        ctx.lineWidth = 1;
        for (let x = bounds.min_x; x <= bounds.max_x; x++) {
            const screenX = offsetX + (x * CELL_SIZE);
            ctx.beginPath();
            ctx.moveTo(screenX, 0);
            ctx.lineTo(screenX, canvas.height);
            ctx.stroke();
        }
        for (let y = bounds.min_y; y <= bounds.max_y; y++) {
            const screenY = offsetY - (y * CELL_SIZE); // Invert Y for screen coords
            ctx.beginPath();
            ctx.moveTo(0, screenY);
            ctx.lineTo(canvas.width, screenY);
            ctx.stroke();
        }

        // Draw connections first (so they appear under rooms)
        roomsAtLevel.forEach(room => {
            const roomX = offsetX + (room.x * CELL_SIZE);
            const roomY = offsetY - (room.y * CELL_SIZE); // Invert Y

            Object.entries(room.exits || {}).forEach(([direction, targetId]) => {
                if (!targetId) return; // Unexplored exit

                const targetRoom = roomsAtLevel.find(r => r.id === targetId);
                if (!targetRoom) return; // Target not at this Z level

                const targetX = offsetX + (targetRoom.x * CELL_SIZE);
                const targetY = offsetY - (targetRoom.y * CELL_SIZE);

                // Draw line
                ctx.strokeStyle = '#16213e';
                ctx.lineWidth = 2;
                ctx.beginPath();
                ctx.moveTo(roomX, roomY);
                ctx.lineTo(targetX, targetY);
                ctx.stroke();
            });
        });

        // Draw rooms
        roomsAtLevel.forEach(room => {
            const roomX = offsetX + (room.x * CELL_SIZE);
            const roomY = offsetY - (room.y * CELL_SIZE); // Invert Y for screen coords

            const isCurrentRoom = room.id === mapData.current_room_id;
            const isSelected = selectedRoom && selectedRoom.id === room.id;

            // Room circle
            ctx.beginPath();
            ctx.arc(roomX, roomY, 8, 0, 2 * Math.PI);

            if (isCurrentRoom) {
                ctx.fillStyle = '#e94560'; // Current room - bright red
            } else if (room.visit_count > 0) {
                ctx.fillStyle = '#4caf50'; // Visited - green
            } else {
                ctx.fillStyle = '#666'; // Unvisited - grey
            }

            ctx.fill();

            if (isSelected) {
                ctx.strokeStyle = '#ffc107';
                ctx.lineWidth = 2;
                ctx.stroke();
            }

            // Draw visit count for frequently visited rooms
            if (room.visit_count > 1) {
                ctx.fillStyle = '#fff';
                ctx.font = '10px Courier New';
                ctx.textAlign = 'center';
                ctx.fillText(room.visit_count.toString(), roomX, roomY + 20);
            }
        });

        // Reset text alignment
        ctx.textAlign = 'left';

    }, [mapData, zLevel, selectedRoom]);

    // Handle canvas click
    const handleCanvasClick = (event) => {
        if (!mapData) return;

        const canvas = canvasRef.current;
        const rect = canvas.getBoundingClientRect();
        const clickX = event.clientX - rect.left;
        const clickY = event.clientY - rect.top;

        const bounds = mapData.bounds;
        const offsetX = GRID_PADDING - (bounds.min_x * CELL_SIZE);
        const offsetY = GRID_PADDING - (bounds.min_y * CELL_SIZE);

        // Find clicked room
        const roomsAtLevel = mapData.rooms.filter(r => r.z === zLevel);

        for (const room of roomsAtLevel) {
            const roomX = offsetX + (room.x * CELL_SIZE);
            const roomY = offsetY - (room.y * CELL_SIZE);

            const distance = Math.sqrt(
                Math.pow(clickX - roomX, 2) + Math.pow(clickY - roomY, 2)
            );

            if (distance <= 10) {
                setSelectedRoom(room);
                return;
            }
        }

        // Clicked empty space
        setSelectedRoom(null);
    };

    return (
        <div className="map-panel">
            <div className="map-header">
                <h3>🗺️ Map</h3>
                {mapData && (
                    <div className="map-stats">
                        <span>Rooms: {mapData.total_rooms}</span>
                        <span className="z-level">
                            Level: {zLevel}
                            <button
                                onClick={() => setZLevel(z => z + 1)}
                                className="z-button"
                                disabled={!mapData || zLevel >= mapData.bounds.max_z}
                            >
                                ▲
                            </button>
                            <button
                                onClick={() => setZLevel(z => z - 1)}
                                className="z-button"
                                disabled={!mapData || zLevel <= mapData.bounds.min_z}
                            >
                                ▼
                            </button>
                        </span>
                    </div>
                )}
            </div>

            <div className="map-canvas-container">
                <canvas
                    ref={canvasRef}
                    width={600}
                    height={400}
                    onClick={handleCanvasClick}
                    className="map-canvas"
                />
            </div>

            {selectedRoom && (
                <div className="room-details">
                    <h4>{selectedRoom.name}</h4>
                    <p className="room-coords">
                        Position: ({selectedRoom.x}, {selectedRoom.y}, {selectedRoom.z})
                    </p>
                    <p className="room-visits">
                        Visited: {selectedRoom.visit_count} {selectedRoom.visit_count === 1 ? 'time' : 'times'}
                    </p>
                    {selectedRoom.exits && Object.keys(selectedRoom.exits).length > 0 && (
                        <div className="room-exits">
                            <strong>Exits:</strong> {Object.keys(selectedRoom.exits).join(', ')}
                        </div>
                    )}
                </div>
            )}

            {!connected && (
                <div className="map-placeholder">
                    <p>Connect to MUD to start mapping</p>
                </div>
            )}

            {connected && mapData && mapData.total_rooms === 0 && (
                <div className="map-placeholder">
                    <p>Explore to build your map</p>
                </div>
            )}
        </div>
    );
}

export default Map;
