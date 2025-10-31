package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"sync"
)

// Channels globais para os managers do servidor
var (
	portalChannel   = make(chan bool)
	mapChannel      = make(chan func(*shared.Jogo))
	gameOverChannel = make(chan struct{})

	// Posição inicial padrão (será lida do mapa)
	defaultPlayerX, defaultPlayerY int
)

// GameService é o objeto que o RPC irá expor
type GameService struct {
	// (Vazio)
}

// Connect adiciona um novo jogador
func (s *GameService) Connect(args *shared.ConnectArgs, reply *shared.ConnectReply) error {
	var wg sync.WaitGroup
	wg.Add(1)

	var newID int

	// Usamos o mapChannel para acessar o estado com segurança
	mapChannel <- func(j *shared.Jogo) {
		defer wg.Done()

		newID = len(j.Players) + 1 // ID simples
		player := shared.PlayerState{
			X:              defaultPlayerX,
			Y:              defaultPlayerY,
			UltimoVisitado: shared.Vazio, // Começa em Vazio
		}
		j.Players[newID] = player

		// Coloca o jogador no mapa
		j.Mapa[player.Y][player.X] = shared.Personagem

		// Prepara a resposta
		reply.PlayerID = newID
		reply.InitialState = *j // Envia uma cópia do estado atual
		log.Printf("Player %d conectou em (%d, %d).", newID, player.X, player.Y)
	}

	wg.Wait() // Espera o mapManager processar a conexão
	return nil
}

// SendInput processa a ação de um jogador
func (s *GameService) SendInput(args *shared.SendInputArgs, reply *shared.SendInputReply) error {
	var wg sync.WaitGroup
	wg.Add(1)

	// Envia a lógica de input para o mapManager
	mapChannel <- func(j *shared.Jogo) {
		defer wg.Done()

		player, ok := j.Players[args.PlayerID]
		if !ok {
			return // Jogador não existe
		}

		if args.Input == "quit" {
			// Remove jogador do mapa
			j.Mapa[player.Y][player.X] = player.UltimoVisitado
			delete(j.Players, args.PlayerID)
			log.Printf("Player %d desconectou.", args.PlayerID)
			return
		}

		dx, dy := 0, 0
		switch args.Input {
		case "w":
			dy = -1
		case "a":
			dx = -1
		case "s":
			dy = 1
		case "d":
			dx = 1
		}

		if dx == 0 && dy == 0 {
			return // Nenhuma ação
		}

		// Lógica de movimento (refatorada)
		if jogoPodeMoverPara(j, player.X+dx, player.Y+dy) {
			_, status := jogoMoverPlayer(j, args.PlayerID, dx, dy)
			reply.StatusMsg = status
		}
	}

	wg.Wait()
	return nil
}

// GetState retorna o estado atual do jogo
func (s *GameService) GetState(args *shared.GetStateArgs, reply *shared.GetStateReply) error {
	var wg sync.WaitGroup
	wg.Add(1)

	mapChannel <- func(j *shared.Jogo) {
		defer wg.Done()
		reply.State = *j // Retorna uma cópia do estado
	}

	wg.Wait()
	return nil
}

func main() {
	// 1. Inicializa o Jogo
	jogo := jogoNovo()
	if err := jogoCarregarMapa("mapa.txt", &jogo); err != nil {
		panic(err)
	}

	// 2. Inicia os Managers (copiados do seu main.go original)
	go mapManager(&jogo)
	go coinManager(&jogo)   // Inicie-os aqui
	go portalManager(&jogo) // Inicie-os aqui
	go patoManager(&jogo)   // Inicie-os aqui

	// 3. Registra e Inicia o Servidor RPC
	rpc.Register(new(GameService))

	listener, err := net.Listen("tcp", ":12345")
	if err != nil {
		log.Fatal("Erro ao ouvir:", err)
	}
	defer listener.Close()

	log.Println("Servidor RPC rodando na porta 12345")
	rpc.Accept(listener)
}

// (Copie seu mapManager, coinManager, portalManager, patoManager para cá)
// O mapManager é o mais importante:
func mapManager(jogo *shared.Jogo) {
	for {
		select {
		case cmd := <-mapChannel:
			cmd(jogo)
		case <-gameOverChannel:
			return
		}
	}
}

// Cria e retorna uma nova instância do jogo
func jogoNovo() shared.Jogo {
	return shared.Jogo{
		Players:            make(map[int]shared.PlayerState),
		PatoUltimoVisitado: shared.Vazio,
	}
}

// Lê um arquivo texto e constrói o mapa
func jogoCarregarMapa(nome string, jogo *shared.Jogo) error {
	// ... (Seu código de jogoCarregarMapa, mas com algumas mudanças)
	arq, err := os.Open(nome)
	if err != nil {
		return err
	}
	defer arq.Close()

	scanner := bufio.NewScanner(arq)
	y := 0
	for scanner.Scan() {
		linha := scanner.Text()
		var linhaElems []shared.Elemento
		runes := []rune(linha)
		for x, ch := range runes {
			e := shared.Vazio
			switch ch {
			case shared.Parede.Simbolo:
				e = shared.Parede
			case shared.Inimigo.Simbolo:
				e = shared.Inimigo
			case shared.Vegetacao.Simbolo:
				e = shared.Vegetacao
			case shared.Personagem.Simbolo:
				// GUARDA A POSIÇÃO INICIAL, MAS NÃO COLOCA O JOGADOR
				defaultPlayerX, defaultPlayerY = x, y
				e = shared.Vazio // Posição fica vazia até alguém conectar
			case shared.Pato.Simbolo:
				jogo.PatoPosX, jogo.PatoPosY = x, y
				jogo.PatoUltimoVisitado = shared.Vazio
				e = shared.Pato
			}
			linhaElems = append(linhaElems, e)
		}
		jogo.Mapa = append(jogo.Mapa, linhaElems)
		y++
	}
	return scanner.Err()
}

// Verifica se PODE mover (mesma lógica)
func jogoPodeMoverPara(jogo *shared.Jogo, x, y int) bool {
	if y < 0 || y >= len(jogo.Mapa) || x < 0 || x >= len(jogo.Mapa[y]) {
		return false
	}
	if jogo.Mapa[y][x].Tangivel {
		return false
	}
	return true
}

// Move o JOGADOR (Lógica refatorada de jogoMoverElemento)
func jogoMoverPlayer(jogo *Jogo, playerID int, dx, dy int) (teleported bool, statusMsg string) {
	// Esta função assume que já foi verificado se pode mover
	player := jogo.Players[playerID] // Pega o estado do jogador
	nx, ny := player.PosX+dx, player.PosY+dy

	elementoNaNovaPosicao := jogo.Mapa[ny][nx]

	switch elementoNaNovaPosicao.simbolo {
	case Moeda.simbolo:
		jogo.Mapa[player.PosY][player.PosX] = player.UltimoVisitado
		player.UltimoVisitado = Vazio
		jogo.Mapa[ny][nx] = Personagem
		player.PosX, player.PosY = nx, ny // Atualiza posição do player
		statusMsg = "Moeda!"
		select {
		case portalChannel <- true:
		default:
		}

	case PortalAtivo.simbolo:
		jogo.Mapa[player.PosY][player.PosX] = player.UltimoVisitado // Libera pos antiga
		jogo.Mapa[ny][nx] = Vazio                                   // Consome portal

		// newX, newY := teleportarJogador(jogo) // (Vc precisa desta função aqui)
		newX, newY := defaultPlayerX, defaultPlayerY // Simples por enquanto
		statusMsg = fmt.Sprintf("Teletransportado para (%d, %d)!", newX, newY)

		player.UltimoVisitado = jogo.Mapa[newY][newX] // Guarda o que está na nova pos
		jogo.Mapa[newY][newX] = Personagem            // Move para nova pos
		player.PosX, player.PosY = newX, newY
		teleported = true

	default:
		jogo.Mapa[player.PosY][player.PosX] = player.UltimoVisitado
		player.UltimoVisitado = elementoNaNovaPosicao
		jogo.Mapa[ny][nx] = Personagem
		player.PosX, player.PosY = nx, ny
		statusMsg = ""
	}

	jogo.Players[playerID] = player // Salva o estado atualizado do jogador
	return
}

// teleportarJogador, patoManager, coinManager, portalManager)
