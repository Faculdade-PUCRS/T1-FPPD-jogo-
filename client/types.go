package main

import "jogo/shared"

// struct de cada "bloco" do mapa
type Elemento struct {
	simbolo  rune
	cor      Cor
	corFundo Cor
	tangivel bool
}

var (
	Personagem       = Elemento{'☺', CorCinzaEscuro, CorPadrao, true}
	PersonagemRemoto = Elemento{'☻', CorVerde, CorPadrao, true} // Outros jogadores
	Inimigo          = Elemento{'☠', CorVermelho, CorPadrao, true}
	Parede           = Elemento{'▤', CorParede, CorFundoParede, true}
	Vegetacao        = Elemento{'♣', CorVerde, CorPadrao, false}
	Vazio            = Elemento{' ', CorPadrao, CorPadrao, false}
	Moeda            = Elemento{'ၜ', CorAmarelo, CorPadrao, false}
	PortalAtivo      = Elemento{'○', CorMagenta, CorPadrao, false}
	PortalInativo    = Vazio
	Pato             = Elemento{'ࠎ', CorAzul, CorPadrao, true}
)

// Jogo
type Jogo struct {
	Mapa               [][]Elemento // grade 2D representando o mapa
	PosX, PosY         int          // posição atual do personagem local
	UltimoVisitado     Elemento     // elemento que estava na posição do personagem antes de mover
	StatusMsg          string       // mensagem para a barra de status
	PatoPosX, PatoPosY int          // posição do pato
	PatoInteragiu      bool         // se o pato foi interagido
	PatoUltimoVisitado Elemento
	PortalAtivo        bool

	Players map[int]shared.PlayerState
}
