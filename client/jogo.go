// client/jogo.go
package main

import (
	"bufio"
	"fmt"
	"jogo/shared"
	"os"
)

// !!! --- CRÍTICO: DEFINIÇÃO DOS CHANNELS --- !!!
// Seus managers (pato, coin, portal, render) precisam que
// estas variáveis sejam definidas em algum lugar no pacote main.
var (
	portalChannel   = make(chan bool)
	mapChannel      = make(chan func(*Jogo))
	gameOverChannel = make(chan struct{})
)

// Cria e retorna uma nova instância do jogo
func jogoNovo() Jogo {
	// O ultimo elemento visitado é inicializado como vazio
	return Jogo{
		UltimoVisitado:     Vazio,
		PatoUltimoVisitado: Vazio,
		Players:            make(map[int]shared.PlayerState), // Inicializa o mapa
	}
}

// Lê um arquivo texto linha por linha e constrói o mapa do jogo
func jogoCarregarMapa(nome string, jogo *Jogo) error {
	arq, err := os.Open(nome)
	if err != nil {
		return err
	}
	defer arq.Close()

	scanner := bufio.NewScanner(arq)
	y := 0
	for scanner.Scan() {
		linha := scanner.Text()
		var linhaElems []Elemento
		runes := []rune(linha)
		for x, ch := range runes {
			e := Vazio
			switch ch {
			case Parede.simbolo:
				e = Parede
			case Inimigo.simbolo:
				e = Inimigo
			case Vegetacao.simbolo:
				e = Vegetacao
			case Personagem.simbolo:
				jogo.PosX, jogo.PosY = x, y // registra a posição inicial do personagem
				// (e = Vazio por padrão)
			case Pato.simbolo:
				jogo.PatoPosX, jogo.PatoPosY = x, y
				jogo.PatoUltimoVisitado = Vazio
				e = Pato
			}
			linhaElems = append(linhaElems, e)
		}
		jogo.Mapa = append(jogo.Mapa, linhaElems)
		y++
	}
	return scanner.Err()
}

// Verifica se o personagem pode se mover para a posição (x, y)
func jogoPodeMoverPara(jogo *Jogo, x, y int) bool {
	if y < 0 || y >= len(jogo.Mapa) || x < 0 || x >= len(jogo.Mapa[y]) {
		return false // Fora do mapa
	}
	if jogo.Mapa[y][x].tangivel {
		return false // Bateu em parede, pato, etc.
	}

	// Verifica se bateu em outro jogador
	for _, player := range jogo.Players {
		if player.PosX == x && player.PosY == y {
			return false
		}
	}
	return true
}

// Move um elemento para a nova posição (LÓGICA ORIGINAL)
// *** Esta função estava faltando no seu 'personagem.go' ***
func jogoMoverElemento(jogo *Jogo, x, y, dx, dy int) bool {
	nx, ny := x+dx, y+dy
	elemento := jogo.Mapa[y][x]
	elementoNaNovaPosicao := jogo.Mapa[ny][nx]

	switch elementoNaNovaPosicao.simbolo {
	case Moeda.simbolo:
		jogo.Mapa[y][x] = jogo.UltimoVisitado // restaura o conteúdo anterior
		jogo.UltimoVisitado = Vazio
		jogo.Mapa[ny][nx] = elemento // move o elemento
		select {
		case portalChannel <- true:
		default:
		}
		return false

	case PortalAtivo.simbolo:
		newX, newY := teleportarJogador(jogo)
		jogo.PatoInteragiu = true
		jogo.StatusMsg = fmt.Sprintf("Teletransportado para (%d, %d)!", newX, newY)
		jogo.Mapa[y][x] = jogo.UltimoVisitado       // restaura o conteúdo anterior
		jogo.Mapa[ny][nx] = Vazio                   // Remove o portal (consumido)
		jogo.UltimoVisitado = jogo.Mapa[newY][newX] // guarda o conteúdo atual
		jogo.Mapa[newY][newX] = elemento            // move o elemento
		jogo.PosX, jogo.PosY = newX, newY           // Atualiza a posição
		return true                                 // indica que houve teletransporte

	default:
		jogo.Mapa[y][x] = jogo.UltimoVisitado   // restaura o conteúdo anterior
		jogo.UltimoVisitado = jogo.Mapa[ny][nx] // guarda o conteúdo atual
		jogo.Mapa[ny][nx] = elemento            // move o elemento
		return false                            // movimento normal
	}
}

// mapManager (Seu código original)
func mapManager(jogo *Jogo) {
	for {
		select {
		case cmd := <-mapChannel:
			cmd(jogo)
		case <-gameOverChannel:
			return
		}
	}
}
