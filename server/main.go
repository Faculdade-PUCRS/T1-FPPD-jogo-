package main

import (
	"jogo/shared"
	"log"
	"maps"
	"net"
	"net/rpc"
	"sync"
)

// ServerState é o único estado do servidor
type ServerState struct {
	mu          sync.Mutex
	players     map[int]shared.PlayerState
	nextID      int
	lastSeqNums map[int]int
}

// GameService implementa os métodos RPC
type GameService struct {
	state *ServerState
}

// Connect registra um novo jogador
func (s *GameService) Connect(args *shared.ConnectArgs, reply *shared.ConnectReply) error {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	newID := s.state.nextID
	s.state.nextID++ // Incrementa para o próximo jogador

	// Posição inicial
	newState := shared.PlayerState{PosX: 1, PosY: 1}
	s.state.players[newID] = newState // Adiciona ao mapa
	s.state.lastSeqNums[newID] = 0    // Inicializa o sequence number

	// Retorna o ID e uma cópia do mapa de jogadores
	reply.PlayerID = newID
	reply.AllPlayers = make(map[int]shared.PlayerState)
	maps.Copy(reply.AllPlayers, s.state.players) // retorna uma cópia dos players atuais

	log.Printf("[RPC] Connect -> ID: %d, Players: %v", newID, reply.AllPlayers)
	return nil
}

// função para atualizar o estado do jogador
func (s *GameService) UpdateState(args *shared.UpdateStateArgs, reply *shared.UpdateStateReply) error {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	// lógica "EXACLY-ONCE"
	lastSeq, ok := s.state.lastSeqNums[args.PlayerID]
	if !ok {
		// Jogador não existe, ignora
		return nil
	}

	// Se o comando for antigo (menor) ou igual ao último processado, ignora
	if args.SequenceNumber <= lastSeq {
		log.Printf("[Seq] Comando %d ignorado (último foi %d)", args.SequenceNumber, lastSeq)
		return nil // Sucesso, mas não faz nada
	}

	// Comando é novo, processa e atualiza
	s.state.lastSeqNums[args.PlayerID] = args.SequenceNumber // Atualiza o último sequence number
	// Atualiza a posição do jogador
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

	log.Printf("[RPC] GetState -> Players: %v", reply.AllPlayers)
	return nil
}

// Disconnect remove um jogador
func (s *GameService) Disconnect(args *shared.DisconnectArgs, reply *shared.DisconnectReply) error {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

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
	delete(s.state.players, args.PlayerID)     // Remove o jogador
	delete(s.state.lastSeqNums, args.PlayerID) // Limpa
	return nil
}

func main() {
	// Inicializa o estado do servidor
	serverState := &ServerState{
		players:     make(map[int]shared.PlayerState),
		nextID:      1,
		lastSeqNums: make(map[int]int),
	}

	// Cria o serviço RPC
	gameService := &GameService{state: serverState}
	// Registra o serviço RPC
	rpc.Register(gameService)

	// abre a porta do servidor
	listener, err := net.Listen("tcp", ":12345")
	if err != nil {
		log.Fatal("Erro ao ouvir:", err)
	}
	defer listener.Close()

	log.Println("Servidor RPC rodando na porta 12345")
	// iniciar o loop de aceitação de conexões
	rpc.Accept(listener)
}
