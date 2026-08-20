package game

// IsKnownCommand reports whether kind is part of the public game command surface.
// Keeping the list next to the domain prevents transport adapters from growing
// their own copy of the command vocabulary.
func IsKnownCommand(kind CommandKind) bool {
	switch kind {
	case CommandSetReady,
		CommandStartMatch,
		CommandAcknowledgeRole,
		CommandResolveOperation,
		CommandSelectOperationTarget,
		CommandOperationExplainDone,
		CommandAdvanceInterlude,
		CommandAdvanceDiscussion,
		CommandSubmitVote,
		CommandContinueResults,
		CommandRematch,
		CommandSetOperationEnabled,
		CommandSetRoleEnabled,
		CommandSetDiscussionTimer,
		CommandSetVirusCount,
		CommandTransferHost,
		CommandKickPlayer:
		return true
	default:
		return false
	}
}
