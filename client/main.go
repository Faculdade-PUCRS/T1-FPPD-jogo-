// client/main.go
package main

import (
	"log"
	"net/rpc"
	"os"
	"sync"
	"time"

	"jogo/shared" // MUDE para o nome do seu módulo
)

var (
	renderChannel   = make(chan struct{})
	client          *rpc.Client
	myID            int
	jogo            Jogo                               // O estado de jogo local é global
	lastServerState = make(map[int]shared.PlayerState) // Último estado vindo do server
	rpcMu           sync.Mutex                         // Protege chamadas RPC
	sequenceNumber  = 0                                // Nosso contador de comandos
	seqMu           sync.Mutex
)

func getNovoSequenceNumber() int {
	seqMu.Lock()
	defer seqMu.Unlock()
	sequenceNumber++
	return sequenceNumber
}

func callWithRetry(serviceMethod string, args interface{}, reply interface{}) {
	// Trava o RPC para não enviar dois comandos ao mesmo tempo
	rpcMu.Lock()
	defer rpcMu.Unlock()

	const maxRetries = 3
	for i := 0; i < maxRetries; i++ {
		err := client.Call(serviceMethod, args, reply)
		if err == nil {
			return // Sucesso!
		}

		// Falha
		log.Printf("Erro RPC (%s): %v. Tentativa %d/%d", serviceMethod, err, i+1, maxRetries)

		// Se for erro de conexão, tenta re-discar
		if err == rpc.ErrShutdown {
			log.Println("Conexão perdida. Tentando reconectar...")
			// (Nota: Em um app real, o 'client' global precisaria
			// ser reatribuído aqui. Para este trabalho, um
			// simples sleep pode ser o suficiente para simular)
			// client, _ = rpc.Dial("tcp", "localhost:12345")
		}

		time.Sleep(500 * time.Millisecond) // Espera antes de tentar de novo
	}
	log.Printf("Falha ao enviar RPC (%s) após %d tentativas.", serviceMethod, maxRetries)
}

func main() {
	// 1. Conecta ao Servidor RPC
	var err error
	client, err = rpc.Dial("tcp", "localhost:12345")
	if err != nil {
		log.Fatal("Erro ao discar:", err)
	}

	// 2. Chama o Connect para entrar no jogo
	connectArgs := &shared.ConnectArgs{}
	connectReply := &shared.ConnectReply{}
	err = client.Call("GameService.Connect", connectArgs, connectReply)
	if err != nil {
		log.Fatal("Erro ao conectar:", err)
	}
	myID = connectReply.PlayerID
	lastServerState = connectReply.AllPlayers
	log.Printf("Conectado. Meu ID: %d", myID)

	// 3. Inicializa a interface
	interfaceIniciar()
	defer interfaceFinalizar()
	defer notifyDisconnect() // Avisa o servidor quando fecharmos

	// 4. Carrega o mapa (como no original)
	mapaFile := "mapa.txt"
	if len(os.Args) > 1 {
		mapaFile = os.Args[1]
	}

	// 5. Inicializa o jogo (como no original)
	jogo = jogoNovo() // 'jogo' é global
	if err := jogoCarregarMapa(mapaFile, &jogo); err != nil {
		panic(err)
	}
	jogo.Players = lastServerState // Seta estado inicial dos players

	// 6. AVISA O SERVIDOR da nossa Posição INICIAL REAL
	// (lida do mapa, em vez da Posição 1,1 padrão)
	go updateServerMyState()

	// 7. Inicia todos os managers LOCAIS (como no original)
	go mapManager(&jogo)
	go coinManager(&jogo)
	go portalManager(&jogo)
	go patoManager(&jogo)
	go renderManager(&jogo) // renderManager modificado

	// 8. Desenha o estado inicial
	interfaceDesenharJogo(&jogo, myID)

	// 9. Loop principal de entrada
	for {
		evento := interfaceLerEventoTeclado()

		// Guarda Posição antiga para checar se houve mudança
		oldX, oldY := jogo.PosX, jogo.PosY

		// Executa a lógica LOCALMENTE
		if continuar := personagemExecutarAcao(evento, &jogo); !continuar {
			break // Sair
		}

		// Se a Posição mudou, avisa o servidor
		if oldX != jogo.PosX || oldY != jogo.PosY {
			go updateServerMyState()
		}

		select {
		case renderChannel <- struct{}{}:
		default:
		}
	}
}

func updateServerMyState() {
	args := &shared.UpdateStateArgs{
		PlayerID:       myID,
		NewX:           jogo.PosX,
		NewY:           jogo.PosY,
		SequenceNumber: getNovoSequenceNumber(),
	}
	reply := &shared.UpdateStateReply{}

	// Usa nossa nova função com reenvio
	callWithRetry("GameService.UpdateState", args, reply)
}

func notifyDisconnect() {
	log.Println("Notificando servidor da desconexão...")
	args := &shared.DisconnectArgs{
		PlayerID:       myID,
		SequenceNumber: getNovoSequenceNumber(),
	}
	reply := &shared.DisconnectReply{}

	// Usa nossa nova função com reenvio
	callWithRetry("GameService.Disconnect", args, reply)
	client.Close()
}

// renderManager (MODIFICADO)
func renderManager(jogo *Jogo) {
	renderTicker := time.NewTicker(100 * time.Millisecond) // ~10 FPS
	defer renderTicker.Stop()

	for {
		select {
		case <-renderTicker.C:
			// 1. Busca estado (GetState não é um comando que modifica,
			// então não precisa de sequenceNumber ou reenvio complexo,
			// mas podemos usar o callWithRetry mesmo assim)
			args := &shared.GetStateArgs{}
			reply := &shared.GetStateReply{}

			// Usamos rpcMu aqui para não colidir com um UpdateState
			rpcMu.Lock()
			err := client.Call("GameService.GetState", args, reply)
			rpcMu.Unlock()

			if err == nil {
				mapChannel <- func(j *Jogo) {
					j.Players = reply.AllPlayers
				}
			} else {
				log.Println("Erro ao buscar estado:", err)
				// O reenvio automático já é coberto pela
				// especificação ("thread dedicada para buscar periodicamente")
				// Se falhar, o próximo tick (100ms) vai tentar de novo.
			}

			// 3. Desenha (com o estado atualizado)
			interfaceDesenharJogo(jogo, myID)

		case <-renderChannel:
			interfaceDesenharJogo(jogo, myID)

		case <-gameOverChannel:
			return
		}
	}
}
