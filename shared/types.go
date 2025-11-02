package shared

// Estado do jogador
type PlayerState struct {
	PosX int
	PosY int
}

// Contrato que o cliente manda para se conectar com o servidor
type ConnectArgs struct{}

// Resposta do servidor ao conectar um novo jogador
type ConnectReply struct {
	PlayerID   int
	AllPlayers map[int]PlayerState // Todos os jogadores, incluindo o novo
}

// Contrato para atualizar o estado do jogador
type UpdateStateArgs struct {
	PlayerID       int
	NewX           int
	NewY           int
	SequenceNumber int
}

// Resposta do servidor à atualização de estado
type UpdateStateReply struct{}

// Contrato para obter o estado de todos os jogadores
type GetStateArgs struct{}

// Resposta do servidor com o estado de todos os jogadores
type GetStateReply struct {
	AllPlayers map[int]PlayerState
}

// Contrato para desconectar um jogador
type DisconnectArgs struct {
	PlayerID       int
	SequenceNumber int
}

// Resposta do servidor à desconexão
type DisconnectReply struct{}
