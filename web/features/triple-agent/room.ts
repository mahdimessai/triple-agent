// Public room/session transport surface. Keep callers insulated from whether a
// capability uses HTTP or WebSocket mechanics.
export {
  createRoom,
  joinRoom,
  leaveRoom,
  leaveRoomOnPageHide,
} from "./transport/room-api";
export {
  connectRoom,
  type RoomSocket,
  type RoomSocketEvent,
} from "./transport/room-socket";
