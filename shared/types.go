// shared/types.go
package shared

// PlayerState é a única coisa que o servidor entende
type PlayerState struct {
	PosX int
	PosY int
}

// === RPC: Connect ===
type ConnectArgs struct{}
type ConnectReply struct {
	PlayerID   int
	AllPlayers map[int]PlayerState // Todos os jogadores, incluindo o novo
}

// === RPC: UpdateState (Cliente-Autoritativo) ===
type UpdateStateArgs struct {
	PlayerID       int
	NewX           int
	NewY           int
	SequenceNumber int
}
type UpdateStateReply struct{}

// === RPC: GetState ===
type GetStateArgs struct{}
type GetStateReply struct {
	AllPlayers map[int]PlayerState
}

// === RPC: Disconnect ===
type DisconnectArgs struct {
	PlayerID       int
	SequenceNumber int
}
type DisconnectReply struct{}
