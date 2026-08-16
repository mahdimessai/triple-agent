/**
 * What the lobby endpoints hand back: the identifiers needed to open a room
 * socket, plus the code to show the table. Everything else about the room -
 * phase, host, roster, settings - arrives over the projection.
 */
export type LobbySession = {
  room_id: string;
  join_code: string;
  player_id: string;
  reconnect_token: string;
};
