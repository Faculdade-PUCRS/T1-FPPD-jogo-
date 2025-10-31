// jogo.go - Funções para manipular os elementos do jogo, como carregar o mapa e mover o personagem
package main

import (
	"bufio"
	"fmt"
	"os"
)

var portalChannel = make(chan bool)
var mapChannel = make(chan func(*Jogo))
var gameOverChannel = make(chan struct{})

// Cria e retorna uma nova instância do jogo
func jogoNovo() Jogo {
	// O ultimo elemento visitado é inicializado como vazio
	// pois o jogo começa com o personagem em uma posição vazia
	return Jogo{UltimoVisitado: Vazio, PatoUltimoVisitado: Vazio}
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
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// Verifica se o personagem pode se mover para a posição (x, y)
func jogoPodeMoverPara(jogo *Jogo, x, y int) bool {
	// Verifica se a coordenada Y está dentro dos limites verticais do mapa
	if y < 0 || y >= len(jogo.Mapa) {
		return false
	}

	// Verifica se a coordenada X está dentro dos limites horizontais do mapa
	if x < 0 || x >= len(jogo.Mapa[y]) {
		return false
	}

	// Verifica se o elemento de destino é tangível (bloqueia passagem)
	if jogo.Mapa[y][x].tangivel {
		return false
	}

	// Pode mover para a posição
	return true
}

// Move um elemento para a nova posição e retorna se houve teletransporte
func jogoMoverElemento(jogo *Jogo, x, y, dx, dy int) bool {
	nx, ny := x+dx, y+dy
	// Obtem elemento atual na posição
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
		// Encontra uma nova posição aleatória para o teletransporte
		newX, newY := teleportarJogador(jogo)
		jogo.PatoInteragiu = true
		jogo.StatusMsg = fmt.Sprintf("Teletransportado para (%d, %d)!", newX, newY)

		jogo.Mapa[y][x] = jogo.UltimoVisitado // restaura o conteúdo anterior

		// Remove o portal (ele é consumido)
		jogo.Mapa[ny][nx] = Vazio

		jogo.UltimoVisitado = jogo.Mapa[newY][newX] // guarda o conteúdo atual da nova posição
		jogo.Mapa[newY][newX] = elemento            // move o elemento

		// Atualiza a posição do jogador
		jogo.PosX, jogo.PosY = newX, newY

		return true // indica que houve teletransporte

	default:
		jogo.Mapa[y][x] = jogo.UltimoVisitado   // restaura o conteúdo anterior
		jogo.UltimoVisitado = jogo.Mapa[ny][nx] // guarda o conteúdo atual da nova posição
		jogo.Mapa[ny][nx] = elemento            // move o elemento
		return false                            // movimento normal
	}
}

func mapManager(Jogo *Jogo) {
	for {
		select {
		case cmd := <-mapChannel:
			cmd(Jogo)
		case <-gameOverChannel:
			return
		}
	}
}
