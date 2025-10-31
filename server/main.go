// server/main.go
package main

import (
	"log"
	"net"
	"net/rpc"
	"sync"

	"jogo/shared" // MUDE para o nome do seu módulo
)

// ServerState é o único estado do servidor
type ServerState struct {
	mu          sync.Mutex
	players     map[int]shared.PlayerState
	nextID      int
	lastSeqNums map[int]int
}

type GameService struct {
	state *ServerState
}

// Connect registra um novo jogador
func (s *GameService) Connect(args *shared.ConnectArgs, reply *shared.ConnectReply) error {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	newID := s.state.nextID
	s.state.nextID++

	// Posição inicial (cliente vai atualizar logo em seguida)
	newState := shared.PlayerState{PosX: 1, PosY: 1}
	s.state.players[newID] = newState
	s.state.lastSeqNums[newID] = 0

	// Retorna o ID e uma cópia do mapa de jogadores
	reply.PlayerID = newID
	reply.AllPlayers = make(map[int]shared.PlayerState)
	for id, pos := range s.state.players {
		reply.AllPlayers[id] = pos
	}

	// Regra 4: Imprimir no terminal
	log.Printf("[RPC] Connect -> ID: %d, Players: %v", newID, reply.AllPlayers)
	return nil
}

// UpdateState aceita cegamente a nova Posição de um cliente
func (s *GameService) UpdateState(args *shared.UpdateStateArgs, reply *shared.UpdateStateReply) error {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	lastSeq, ok := s.state.lastSeqNums[args.PlayerID]
	if !ok {
		// Jogador não existe, ignora
		return nil
	}

	// Se o comando for antigo (menor) ou igual ao último processado,
	// apenas ignore. Não retorne erro, pois o cliente
	// pode estar reenviando e só precisa de um OK.
	if args.SequenceNumber <= lastSeq {
		log.Printf("[Seq] Comando %d ignorado (último foi %d)", args.SequenceNumber, lastSeq)
		return nil // Sucesso, mas não faz nada
	}

	// Comando é novo, processa e atualiza
	s.state.lastSeqNums[args.PlayerID] = args.SequenceNumber
	s.state.players[args.PlayerID] = shared.PlayerState{
		PosX: args.NewX,
		PosY: args.NewY,
	}
	return nil
}

// GetState envia a lista de Posições para o cliente
func (s *GameService) GetState(args *shared.GetStateArgs, reply *shared.GetStateReply) error {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	// Retorna uma cópia do mapa
	reply.AllPlayers = make(map[int]shared.PlayerState)
	for id, pos := range s.state.players {
		reply.AllPlayers[id] = pos
	}

	// Regra 4: Imprimir resposta
	log.Printf("[RPC] GetState -> Players: %v", reply.AllPlayers)
	return nil
}

// Disconnect remove um jogador
func (s *GameService) Disconnect(args *shared.DisconnectArgs, reply *shared.DisconnectReply) error {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	// Regra 4: Imprimir requisição
	log.Printf("[RPC] Disconnect <- ID: %d", args.PlayerID)

	lastSeq, ok := s.state.lastSeqNums[args.PlayerID]
	if !ok {
		return nil // Jogador já saiu
	}
	if args.SequenceNumber <= lastSeq {
		log.Printf("[Seq] Disconnect %d ignorado (último foi %d)", args.SequenceNumber, lastSeq)
		return nil
	}

	s.state.lastSeqNums[args.PlayerID] = args.SequenceNumber
	delete(s.state.players, args.PlayerID)
	delete(s.state.lastSeqNums, args.PlayerID) // Limpa
	return nil
}

func main() {
	serverState := &ServerState{
		players:     make(map[int]shared.PlayerState),
		nextID:      1,
		lastSeqNums: make(map[int]int),
	}

	gameService := &GameService{state: serverState}
	rpc.Register(gameService)

	listener, err := net.Listen("tcp", ":12345")
	if err != nil {
		log.Fatal("Erro ao ouvir:", err)
	}
	defer listener.Close()

	log.Println("Servidor RPC (Burro) rodando na porta 12345")
	rpc.Accept(listener)
}
