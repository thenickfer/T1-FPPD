// Código exemplo para o trabaho de sistemas distribuidos (eleicao em anel)
// By Cesar De Rose - 2022

package main

import (
	"fmt"
	"sync"
)

type mensagem struct {
	tipo  int    // tipo da mensagem para fazer o controle do que fazer (eleição, confirmacao da eleicao)
	corpo [3]int // conteudo da mensagem para colocar os ids (usar um tamanho ocmpativel com o numero de processos no anel)
}

var (
	chans = []chan mensagem{ // vetor de canias para formar o anel de eleicao - chan[0], chan[1] and chan[2] ...
		make(chan mensagem),
		make(chan mensagem),
		make(chan mensagem),
		make(chan mensagem),
	}
	controle = make(chan int)
	wg       sync.WaitGroup // wg is used to wait for the program to finish
)

func ElectionControler(in chan int) {
	defer wg.Done()

	var temp mensagem
	var leader int

	// comandos para o anel iciam aqui

	// mudar o processo 0 - canal de entrada 3 - para falho (defini mensagem tipo 2 pra isto)

	for i := 0; i < 10; i++ {
		var curr int = i % 4

		temp.tipo = 2
		fmt.Printf("Controle: mudar o processo %d para falho\n", curr)
		chans[3-curr] <- temp

		fmt.Printf("Controle: confirmação %d\n", <-in) // receber e imprimir confirmação
		fmt.Printf("Controle: iniciar votacao no processo %d\n", (curr+1)%4)
		temp.tipo = 4
		chans[3-(curr+1)%4] <- temp
		var response int = <-in
		for response == -1 {
			fmt.Printf("Controle: erro na votacao no processo %d, recomecando\n", (curr+1)%4)
			chans[3-(curr+1)%4] <- temp
			response = <-in
		}

		leader = response
		fmt.Printf("Controle: votacao no processo %d concluida, lider %d\n", curr, leader)

		var antecedente int = curr - 1
		if antecedente < 0 {
			antecedente = 3
		}

		fmt.Printf("Controle: reativar processo %d\n", antecedente)
		temp.tipo = 3
		chans[3-antecedente] <- temp
		<-in
		fmt.Printf("Controle: processo %d reativado, começando eleição\n", antecedente)
		temp.tipo = 4
		chans[3-antecedente] <- temp
		response = <-in
		for response == -1 {
			fmt.Printf("Controle: erro na votacao no processo %d, recomecando\n", antecedente)
			chans[3-antecedente] <- temp
			response = <-in
		}
		fmt.Printf("Controle: processo %d eleito\n", response)

	}

	fmt.Printf("Controle: testando com 2+ falhas\n")

	fmt.Printf("Controle: mudar o processo 1 para falho\n")

	temp.tipo = 2
	chans[0] <- temp

	fmt.Printf("Controle: confirmação %d\n", <-in) // receber e imprimir confirmação
	fmt.Printf("Controle: iniciar votacao no processo 0\n")
	temp.tipo = 4
	chans[3] <- temp
	var response int = <-in
	for response == -1 {
		fmt.Printf("Controle: erro na votacao no processo 0, recomecando\n")
		chans[3] <- temp
		response = <-in
	}

	leader = response
	fmt.Printf("Controle: votacao no processo 0 concluida, lider %d\n", leader)

	fmt.Printf("Controle: mudar o processo 3 para falho\n")
	temp.tipo = 2
	chans[2] <- temp

	fmt.Printf("Controle: confirmação %d\n", <-in) // receber e imprimir confirmação
	fmt.Printf("Controle: iniciar votacao no processo 0\n")
	temp.tipo = 4
	chans[3] <- temp
	response = <-in
	for response == -1 {
		fmt.Printf("Controle: erro na votacao no processo 0, recomecando\n")
		chans[3] <- temp
		response = <-in
	}

	leader = response
	fmt.Printf("Controle: votacao no processo 0 concluida, lider %d\n", leader)

	// matar os outrs processos com mensagens não conhecidas (só pra cosumir a leitura)

	temp.tipo = 10
	for i := 0; i < 4; i++ {
		chans[i] <- temp
	}

	fmt.Println("\n   Processo controlador concluído\n")
}

func ElectionStage(TaskId int, in chan mensagem, out chan mensagem, leader int) {
	defer wg.Done()

	// variaveis locais que indicam se este processo é o lider e se esta ativo

	var actualLeader int
	var bFailed bool = false // todos inciam sem falha

	actualLeader = leader // indicação do lider veio por parâmatro
	for {
		temp := <-in // ler mensagem
		fmt.Printf("%2d: recebi mensagem %d, [ %d, %d, %d ]\n", TaskId, temp.tipo, temp.corpo[0], temp.corpo[1], temp.corpo[2])

		switch temp.tipo {
		case 2:
			{
				bFailed = true
				fmt.Printf("%2d: falho %v \n", TaskId, bFailed)
				fmt.Printf("%2d: lider atual %d\n", TaskId, actualLeader)
				controle <- -5
			}
		case 3:
			{
				bFailed = false
				fmt.Printf("%2d: falho %v \n", TaskId, bFailed)
				fmt.Printf("%2d: lider atual %d\n", TaskId, actualLeader)
				controle <- -5
			}
		case 4:
			{
				fmt.Printf("%2d: disparando eleicao\n", TaskId)
				temp.tipo = 5
				temp.corpo = [3]int{-1, -1, -1}
				out <- temp
				temp := <-in
				if temp.tipo == 5 {
					fmt.Printf("%2d: Recebeu votacao\n", TaskId)
					actualLeader = max(TaskId, temp.corpo[0], temp.corpo[1], temp.corpo[2])
					temp.tipo = 6
					temp.corpo = [3]int{actualLeader, -1, -1}
					fmt.Printf("%2d: Mandando confirmacao\n", TaskId)
					out <- temp
					temp := <-in
					if temp.tipo == 6 {
						fmt.Printf("%2d: Votacao concluida, lider %d\n", TaskId, actualLeader)
						controle <- actualLeader
					} else {
						fmt.Printf("%2d: Erro na votacao\n", TaskId)
						controle <- -1
					}
				}
			}
		case 5:
			{
				if bFailed == true {
					out <- temp
					break
				}
				fmt.Printf("%2d: votacao\n", TaskId)
				for i := 0; i < 3; i++ {
					if temp.corpo[i] == -1 {
						temp.corpo[i] = TaskId
						break
					}
				}
				out <- temp
			}
		case 6:
			{
				if bFailed == true {
					out <- temp
					break
				}
				actualLeader = temp.corpo[0]
				fmt.Printf("%2d: confirmando, lider %d\n", TaskId, actualLeader)
				out <- temp
			}
		default:
			{
				fmt.Printf("%2d: não conheço este tipo de mensagem\n", TaskId)
				fmt.Printf("%2d: lider atual %d\n", TaskId, actualLeader)
				return
			}
		}
	}

	fmt.Printf("%2d: terminei \n", TaskId)
}

func main() {

	wg.Add(5) // Add a count of four, one for each goroutine

	// criar os processo do anel de eleicao

	go ElectionStage(0, chans[3], chans[0], 0) // este é o lider
	go ElectionStage(1, chans[0], chans[1], 0) // não é lider, é o processo 0
	go ElectionStage(2, chans[1], chans[2], 0) // não é lider, é o processo 0
	go ElectionStage(3, chans[2], chans[3], 0) // não é lider, é o processo 0

	fmt.Println("\n   Anel de processos criado")

	// criar o processo controlador

	go ElectionControler(controle)

	fmt.Println("\n   Processo controlador criado\n")

	wg.Wait() // Wait for the goroutines to finish\
}
