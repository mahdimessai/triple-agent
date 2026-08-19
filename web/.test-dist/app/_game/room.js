"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createRoom = createRoom;
exports.joinRoom = joinRoom;
exports.leaveRoom = leaveRoom;
exports.leaveRoomOnPageHide = leaveRoomOnPageHide;
exports.connectRoom = connectRoom;
const protocol_1 = require("./protocol");
const DEFAULT_API_BASE_URL = "http://localhost:8080";
function apiBaseUrl() {
    return (process.env.NEXT_PUBLIC_TRIPLE_AGENT_API_URL ?? DEFAULT_API_BASE_URL).replace(/\/$/, "");
}
function apiUrl(path) {
    return `${apiBaseUrl()}${path}`;
}
async function postJson(path, body) {
    const response = await fetch(apiUrl(path), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
    });
    let payload;
    try {
        payload = await response.json();
    }
    catch {
        payload = null;
    }
    if (!response.ok) {
        const message = typeof payload === "object"
            && payload !== null
            && "error" in payload
            && typeof payload.error === "string"
            ? payload.error
            : `Request failed (${response.status})`;
        throw new Error(message);
    }
    return payload;
}
async function roomIdentityRequest(path, body) {
    const payload = await postJson(path, body);
    if (!(0, protocol_1.isRoomIdentity)(payload))
        throw new Error("The room server returned an invalid session");
    return payload;
}
function createRoom(playerName) {
    return roomIdentityRequest("/api/lobbies", { player_name: playerName });
}
function joinRoom(joinCode, playerName) {
    return roomIdentityRequest("/api/lobbies/join", {
        join_code: joinCode,
        player_name: playerName,
    });
}
async function leaveRoom(identity) {
    await postJson("/api/lobbies/leave", {
        room_id: identity.room_id,
        player_id: identity.player_id,
        reconnect_token: identity.reconnect_token,
    });
}
function leaveRoomOnPageHide(identity) {
    const body = JSON.stringify({
        room_id: identity.room_id,
        player_id: identity.player_id,
        reconnect_token: identity.reconnect_token,
    });
    const url = apiUrl("/api/lobbies/leave");
    if (typeof navigator !== "undefined" && typeof navigator.sendBeacon === "function") {
        const accepted = navigator.sendBeacon(url, new Blob([body], { type: "text/plain;charset=UTF-8" }));
        if (accepted)
            return;
    }
    void fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body,
        keepalive: true,
    });
}
function websocketUrl(identity) {
    const url = new URL(apiBaseUrl());
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    url.pathname = "/ws";
    url.search = new URLSearchParams({
        room_id: identity.room_id,
        player_id: identity.player_id,
    }).toString();
    return url.toString();
}
function nextRequestId() {
    return crypto.randomUUID();
}
function serializeCommand(command, expectedVersion, requestId) {
    return {
        ...command,
        request_id: requestId,
        expected_version: expectedVersion,
    };
}
function closeDetails(error) {
    return {
        terminal: error.status === 401 || error.status === 410,
        status: error.status,
        message: error.error,
        ...(error.code ? { code: error.code } : {}),
    };
}
function connectRoom(identity, onEvent) {
    const socket = new WebSocket(websocketUrl(identity));
    let disposed = false;
    let closed = false;
    function emit(event) {
        if (!disposed)
            onEvent(event);
    }
    function emitClosed(details) {
        if (disposed || closed)
            return;
        closed = true;
        onEvent({ type: "closed", ...details });
    }
    socket.addEventListener("open", () => {
        if (disposed)
            return;
        socket.send(JSON.stringify({ type: "room.auth", reconnect_token: identity.reconnect_token }));
    });
    socket.addEventListener("message", (event) => {
        if (disposed)
            return;
        let parsed;
        try {
            parsed = JSON.parse(String(event.data));
        }
        catch {
            emit({ type: "invalid-message" });
            return;
        }
        const message = (0, protocol_1.parseRoomServerMessage)(parsed);
        if (!message) {
            emit({ type: "invalid-message" });
            return;
        }
        if (message.type === "session.authenticated") {
            emit({ type: "open" });
            return;
        }
        emit({ type: "message", message });
        if (message.type === "session.error") {
            emitClosed(closeDetails(message));
            socket.close();
        }
    });
    socket.addEventListener("error", () => {
        emitClosed({ terminal: false, message: "The room connection failed" });
        socket.close();
    });
    socket.addEventListener("close", () => {
        emitClosed({ terminal: false });
    });
    return {
        send(command, expectedVersion) {
            if (disposed || socket.readyState !== WebSocket.OPEN) {
                throw new Error("Room connection is not open");
            }
            const requestId = nextRequestId();
            socket.send(JSON.stringify(serializeCommand(command, expectedVersion, requestId)));
            return requestId;
        },
        resync() {
            if (disposed || socket.readyState !== WebSocket.OPEN)
                return;
            socket.send(JSON.stringify({ kind: "room.resync" }));
        },
        close() {
            if (disposed)
                return;
            disposed = true;
            socket.close();
        },
    };
}
