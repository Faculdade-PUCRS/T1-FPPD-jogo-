package main

import (
	"math/rand"
	"time"
)

func portalManager(jogo *Jogo) {
	var posicaoPortalX, posicaoPortalY int

	for {
		select {
		case ativar := <-portalChannel:
			if ativar {
				cmd := func(jogo *Jogo) {
					if !jogo.PortalAtivo {
						var x, y int
						jogo.PortalAtivo, x, y = ativarPortal(jogo)
						posicaoPortalX, posicaoPortalY = x, y

						go func() {
							time.Sleep(15 * time.Second)
							portalChannel <- false
						}()
					}
				}
				mapChannel <- cmd
			} else {
				cmd := func(jogo *Jogo) {
					if jogo.PortalAtivo {
						clearPortal(jogo, posicaoPortalX, posicaoPortalY)
						jogo.PortalAtivo = false
					}
				}
				mapChannel <- cmd
			}
		case <-gameOverChannel:
			return
		}
	}
}

func ativarPortal(jogo *Jogo) (bool, int, int) {

	maxY := len(jogo.Mapa)
	maxX := len(jogo.Mapa[0])

	for {
		// Gera uma posição aleatória para x e y
		x := rand.Intn(maxX)
		y := rand.Intn(maxY)

		// Verifica se a posição é válida (vazio e não tangível)
		if !jogo.Mapa[y][x].tangivel && jogo.Mapa[y][x].simbolo == Vazio.simbolo {

			jogo.Mapa[y][x] = PortalAtivo
			jogo.PatoInteragiu = false
			return true, x, y
		}
	}
}

func clearPortal(jogo *Jogo, x int, y int) {
	// Verifica se ainda há um portal na posição
	if jogo.Mapa[y][x].simbolo == PortalAtivo.simbolo {
		jogo.Mapa[y][x] = PortalInativo
		jogo.PatoInteragiu = true
	}
}

func teleportarJogador(jogo *Jogo) (int, int) {
	maxY := len(jogo.Mapa)
	maxX := len(jogo.Mapa[0])

	for {
		x := rand.Intn(maxX)
		y := rand.Intn(maxY)

		if !jogo.Mapa[y][x].tangivel && jogo.Mapa[y][x].simbolo == Vazio.simbolo {
			return x, y
		}
	}
}
