package main

import (
	"log"
	"net/rpc"
	"os"
	"sync"
	"time"

	"jogo/shared"
)

var (
	renderChannel   = make(chan struct{})              // Sinaliza para renderManager redesenhar
	client          *rpc.Client                        // Conexão RPC com o servidor
	myID            int                                // Nosso ID de jogador
	jogo            Jogo                               // O estado de jogo local é global
	lastServerState = make(map[int]shared.PlayerState) // Último estado vindo do server
	rpcMu           sync.Mutex                         // Protege chamadas RPC
	sequenceNumber  = 0                                // Contador de comandos
	seqMu           sync.Mutex                         // Protege sequenceNumber garantindo execução atômica
)

// garante que cada chamada RPC que modifica estado
// use um sequence number único
func getNovoSequenceNumber() int {
	seqMu.Lock()
	defer seqMu.Unlock()
	sequenceNumber++
	return sequenceNumber
}

// função genérica para chamadas RPC com reenvio
func callWithRetry(serviceMethod string, args interface{}, reply interface{}) {
	// Trava o RPC para não enviar dois comandos ao mesmo tempo
	rpcMu.Lock()
	defer rpcMu.Unlock()

	// Tenta até 3 vezes
	const maxRetries = 3
	for i := range maxRetries {

		// Se em alguma tentativa retornar com erro nil, retorna sucesso
		err := client.Call(serviceMethod, args, reply)
		if err == nil {
			return
		}

		// loga o erro
		log.Printf("Erro RPC (%s): %v. Tentativa %d/%d", serviceMethod, err, i+1, maxRetries)

		time.Sleep(500 * time.Millisecond) // Espera antes de tentar de novo
	}
	log.Printf("Falha ao enviar RPC (%s) após %d tentativas.", serviceMethod, maxRetries)
}

func main() {
	var err error
	// Conecta ao servidor RPC
	client, err = rpc.Dial("tcp", "localhost:12345")
	if err != nil {
		log.Fatal("Erro ao conectar:", err)
	}

	// Chama o Connect para entrar no jogo
	connectArgs := &shared.ConnectArgs{}
	connectReply := &shared.ConnectReply{}
	err = client.Call("GameService.Connect", connectArgs, connectReply)
	if err != nil {
		log.Fatal("Erro ao conectar:", err)
	}
	// Servidor retornou nosso ID e a lista de jogadores
	myID = connectReply.PlayerID
	lastServerState = connectReply.AllPlayers
	log.Printf("Conectado. ID: %d", myID)

	// Inicializa a interface (termbox)
	interfaceIniciar()
	defer interfaceFinalizar()
	defer notifyDisconnect() // Avisa o servidor quando fecharmos

	// Usa "mapa.txt" como arquivo padrão ou lê o primeiro argumento
	mapaFile := "mapa.txt"
	if len(os.Args) > 1 {
		mapaFile = os.Args[1]
	}

	// Inicializa o jogo
	jogo = jogoNovo() // 'jogo' é global
	if err := jogoCarregarMapa(mapaFile, &jogo); err != nil {
		panic(err)
	}
	jogo.Players = lastServerState // Seta estado inicial dos players

	// goroutine para atualizar o servidor quando mudarmos de estado
	go updateServerMyState()

	// 7. Inicia todos os managers LOCAIS (como no original)
	go mapManager(&jogo)
	go coinManager(&jogo)
	go portalManager(&jogo)
	go patoManager(&jogo)
	go renderManager(&jogo)

	// 8. Desenha o estado inicial
	interfaceDesenharJogo(&jogo)

	// 9. Loop principal de entrada
	for {
		evento := interfaceLerEventoTeclado()

		// Guarda Posição antiga para checar se houve mudança
		oldX, oldY := jogo.PosX, jogo.PosY

		if continuar := personagemExecutarAcao(evento, &jogo); !continuar {
			break
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

// atualiza o estado do jogador no servidor
func updateServerMyState() {
	// Prepara argumentos
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

// notifica o servidor que estamos saindo do jogo
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

// renderManager
func renderManager(jogo *Jogo) {
	// Timer para render automático a cada 100ms
	renderTicker := time.NewTicker(100 * time.Millisecond)
	defer renderTicker.Stop()

	for {
		select {
		case <-renderTicker.C:
			// A cada tick, busca estado do servidor
			args := &shared.GetStateArgs{}
			reply := &shared.GetStateReply{}

			// Usamos o mutex para proteger a chamada RPC
			rpcMu.Lock()
			err := client.Call("GameService.GetState", args, reply)
			rpcMu.Unlock()

			// Se a chamada foi bem sucedida]
			if err == nil {
				// Atualiza o estado local de todos os players
				mapChannel <- func(j *Jogo) {
					j.Players = reply.AllPlayers
				}
			}

			// Desenha com o estado atualizado
			interfaceDesenharJogo(jogo)

		// render para quando nos movemos
		case <-renderChannel:
			interfaceDesenharJogo(jogo)

		// finaliza quando o jogo acabar
		case <-gameOverChannel:
			return
		}
	}
}
