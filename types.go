package main

type Elemento struct {
	simbolo  rune
	cor      Cor
	corFundo Cor
	tangivel bool // Indica se o elemento bloqueia passagem
}

var (
	Personagem    = Elemento{'☺', CorCinzaEscuro, CorPadrao, true}
	Inimigo       = Elemento{'☠', CorVermelho, CorPadrao, true}
	Parede        = Elemento{'▤', CorParede, CorFundoParede, true}
	Vegetacao     = Elemento{'♣', CorVerde, CorPadrao, false}
	Vazio         = Elemento{' ', CorPadrao, CorPadrao, false}
	Moeda         = Elemento{'ၜ', CorAmarelo, CorPadrao, false}
	PortalAtivo   = Elemento{'○', CorMagenta, CorPadrao, false}
	PortalInativo = Vazio
	Pato          = Elemento{'ࠎ', CorAzul, CorPadrao, true}
)

type PlayerState struct {
	PosX, PosY     int      // posição atual do personagem
	UltimoVisitado Elemento // elemento que estava na posição do personagem antes de mover
}

type Jogo struct {
	Mapa               [][]Elemento        // grade 2D representando o mapa
	Players            map[int]PlayerState // lista de estados dos jogadores
	StatusMsg          string              // mensagem para a barra de status
	PatoPosX, PatoPosY int                 // posição do pato
	PatoInteragiu      bool                // se o pato foi interagido
	PatoUltimoVisitado Elemento
	PortalAtivo        bool
}

type ConnectArgs struct {
	// Vazio por enquanto
}
type ConnectReply struct {
	PlayerID     int
	InitialState Jogo
}

// SendInput
type SendInputArgs struct {
	PlayerID int
	Input    string // "w", "a", "s", "d", "quit"
}
type SendInputReply struct {
	StatusMsg string // O servidor pode enviar uma msg de status de volta
}

// GetState
type GetStateArgs struct {
	// Vazio
}
type GetStateReply struct {
	State Jogo
}
