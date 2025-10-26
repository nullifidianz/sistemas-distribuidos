//go:build !client
// +build !client

package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/pebbe/zmq4"
	msgpack "github.com/vmihailenco/msgpack/v5"
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type botMessage struct {
	Service string      `msgpack:"service"`
	Data    interface{} `msgpack:"data"`
}

type bot struct {
	reqSocket    *zmq4.Socket
	subSocket    *zmq4.Socket
	context      *zmq4.Context
	logicalClock int
	username     string
	messageCount int
	channels     []string
}

func newBot() *bot {
	context, err := zmq4.NewContext()
	if err != nil {
		log.Fatal("Erro ao criar contexto ZMQ:", err)
	}

	reqSocket, err := context.NewSocket(zmq4.REQ)
	if err != nil {
		log.Fatal("Erro ao criar socket REQ:", err)
	}

	subSocket, err := context.NewSocket(zmq4.SUB)
	if err != nil {
		log.Fatal("Erro ao criar socket SUB:", err)
	}

	// Gerar nome aleatório
	usernames := []string{
		"BotAlpha", "BotBeta", "BotGamma", "BotDelta", "BotEpsilon",
		"BotZeta", "BotEta", "BotTheta", "BotIota", "BotKappa",
		"BotLambda", "BotMu", "BotNu", "BotXi", "BotOmicron",
		"BotPi", "BotRho", "BotSigma", "BotTau", "BotUpsilon",
	}

	username := usernames[rand.Intn(len(usernames))] + fmt.Sprintf("%d", rand.Intn(1000))

	return &bot{
		reqSocket:    reqSocket,
		subSocket:    subSocket,
		context:      context,
		logicalClock: 0,
		username:     username,
		messageCount: 0,
		channels:     []string{},
	}
}

func (b *bot) Connect() error {
	// Conectar ao broker
	err := b.reqSocket.Connect("tcp://broker:5555")
	if err != nil {
		return fmt.Errorf("erro ao conectar ao broker: %v", err)
	}

	// Conectar ao proxy
	err = b.subSocket.Connect("tcp://proxy:5558")
	if err != nil {
		return fmt.Errorf("erro ao conectar ao proxy: %v", err)
	}

	// Subscrever a todas as mensagens
	err = b.subSocket.SetSubscribe("")
	if err != nil {
		return fmt.Errorf("erro ao configurar subscription: %v", err)
	}

	fmt.Printf("Bot '%s' conectado ao sistema\n", b.username)
	return nil
}

func (b *bot) Close() {
	b.reqSocket.Close()
	b.subSocket.Close()
	b.context.Term()
}

func (b *bot) incrementClock() int {
	b.logicalClock++
	return b.logicalClock
}

func (b *bot) updateClock(receivedClock int) {
	b.logicalClock = max(b.logicalClock, receivedClock) + 1
}

func (b *bot) sendRequest(service string, data interface{}) (interface{}, error) {
	// Incrementar relógio lógico antes de enviar
	clock := b.incrementClock()

	// Adicionar clock aos dados se for um mapa
	if dataMap, ok := data.(map[string]interface{}); ok {
		dataMap["clock"] = clock
		data = dataMap
	}

	request := botMessage{
		Service: service,
		Data:    data,
	}

	// Serializar mensagem
	encoded, err := msgpack.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar mensagem: %v", err)
	}

	// Enviar requisição
	_, err = b.reqSocket.SendBytes(encoded, 0)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar mensagem: %v", err)
	}

	// Receber resposta
	responseBytes, err := b.reqSocket.RecvBytes(0)
	if err != nil {
		return nil, fmt.Errorf("erro ao receber resposta: %v", err)
	}

	// Deserializar resposta
	var response botMessage
	err = msgpack.Unmarshal(responseBytes, &response)
	if err != nil {
		return nil, fmt.Errorf("erro ao deserializar resposta: %v", err)
	}

	// Atualizar relógio lógico se recebeu clock
	if responseData, ok := response.Data.(map[string]interface{}); ok {
		if receivedClock, exists := responseData["clock"]; exists {
			if clockVal, ok := receivedClock.(int); ok {
				b.updateClock(clockVal)
			}
		}
	}

	return response.Data, nil
}

func (b *bot) Login() error {
	data := map[string]interface{}{
		"user":      b.username,
		"timestamp": time.Now().UnixMilli(),
	}

	response, err := b.sendRequest("login", data)
	if err != nil {
		return err
	}

	responseData, ok := response.(map[string]interface{})
	if !ok {
		return fmt.Errorf("resposta inválida")
	}

	status, _ := responseData["status"].(string)
	if status == "erro" {
		description, _ := responseData["description"].(string)
		return fmt.Errorf("erro no login: %s", description)
	}

	fmt.Printf("Bot '%s' logado com sucesso\n", b.username)
	return nil
}

func (b *bot) ListChannels() ([]string, error) {
	data := map[string]interface{}{
		"timestamp": time.Now().UnixMilli(),
	}

	response, err := b.sendRequest("channels", data)
	if err != nil {
		return nil, err
	}

	responseData, ok := response.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("resposta inválida")
	}

	channels, _ := responseData["channels"].([]interface{})
	var channelList []string
	for _, channel := range channels {
		if channelName, ok := channel.(string); ok {
			channelList = append(channelList, channelName)
		}
	}

	return channelList, nil
}

func (b *bot) CreateChannel(channelName string) error {
	data := map[string]interface{}{
		"channel":   channelName,
		"timestamp": time.Now().UnixMilli(),
	}

	response, err := b.sendRequest("channel", data)
	if err != nil {
		return err
	}

	responseData, ok := response.(map[string]interface{})
	if !ok {
		return fmt.Errorf("resposta inválida")
	}

	status, _ := responseData["status"].(string)
	if status == "erro" {
		description, _ := responseData["description"].(string)
		return fmt.Errorf("erro ao criar canal: %s", description)
	}

	fmt.Printf("Bot '%s' criou canal '%s'\n", b.username, channelName)
	return nil
}

func (b *bot) PublishMessage(channel, message string) error {
	data := map[string]interface{}{
		"user":      b.username,
		"channel":   channel,
		"message":   message,
		"timestamp": time.Now().UnixMilli(),
	}

	response, err := b.sendRequest("publish", data)
	if err != nil {
		return err
	}

	responseData, ok := response.(map[string]interface{})
	if !ok {
		return fmt.Errorf("resposta inválida")
	}

	status, _ := responseData["status"].(string)
	if status == "erro" {
		errorMsg, _ := responseData["message"].(string)
		return fmt.Errorf("erro ao publicar: %s", errorMsg)
	}

	return nil
}

func (b *bot) ListenForMessages() {
	go func() {
		for {
			// Receber tópico e mensagem
			_, err := b.subSocket.RecvBytes(0) // tópico
			if err != nil {
				log.Printf("Erro ao receber mensagem: %v", err)
				continue
			}

			messageBytes, err := b.subSocket.RecvBytes(0)
			if err != nil {
				log.Printf("Erro ao receber dados da mensagem: %v", err)
				continue
			}

			// Deserializar mensagem
			var message botMessage
			err = msgpack.Unmarshal(messageBytes, &message)
			if err != nil {
				log.Printf("Erro ao deserializar mensagem: %v", err)
				continue
			}

			// Atualizar relógio lógico
			if messageData, ok := message.Data.(map[string]interface{}); ok {
				if receivedClock, exists := messageData["clock"]; exists {
					if clockVal, ok := receivedClock.(int); ok {
						b.updateClock(clockVal)
					}
				}
			}

			// Processar mensagem baseada no serviço
			switch message.Service {
			case "publication":
				if messageData, ok := message.Data.(map[string]interface{}); ok {
					user, _ := messageData["user"].(string)
					channel, _ := messageData["channel"].(string)
					msg, _ := messageData["message"].(string)
					fmt.Printf("[%s] Bot '%s' recebeu: %s: %s\n", channel, b.username, user, msg)
				}
			case "private_message":
				if messageData, ok := message.Data.(map[string]interface{}); ok {
					src, _ := messageData["src"].(string)
					dst, _ := messageData["dst"].(string)
					msg, _ := messageData["message"].(string)
					if dst == b.username {
						fmt.Printf("[PRIVADO] Bot '%s' recebeu de %s: %s\n", b.username, src, msg)
					}
				}
			}
		}
	}()
}

func (b *bot) Run() {
	// Fazer login
	err := b.Login()
	if err != nil {
		log.Fatal("Erro no login:", err)
	}

	// Iniciar escuta de mensagens
	b.ListenForMessages()

	// Mensagens pré-definidas
	messages := []string{
		"Olá pessoal!",
		"Como vocês estão?",
		"Alguém quer conversar?",
		"Que dia bonito hoje!",
		"Estou testando o sistema",
		"Funcionando perfeitamente!",
		"ZeroMQ é incrível!",
		"Sistemas distribuídos são fascinantes",
		"Vamos fazer mais testes?",
		"Até a próxima mensagem!",
	}

	// Loop principal
	for {
		// Atualizar lista de canais
		availableChannels, err := b.ListChannels()
		if err != nil {
			log.Printf("Erro ao listar canais: %v", err)
		} else {
			b.channels = availableChannels
		}

		// Se não há canais, criar um
		if len(b.channels) == 0 {
			channelName := fmt.Sprintf("canal%d", rand.Intn(1000))
			err := b.CreateChannel(channelName)
			if err != nil {
				log.Printf("Erro ao criar canal: %v", err)
			} else {
				b.channels = append(b.channels, channelName)
			}
		}

		// Escolher canal aleatório
		selectedChannel := b.channels[rand.Intn(len(b.channels))]

		// Enviar 10 mensagens
		for i := 0; i < 10; i++ {
			message := messages[rand.Intn(len(messages))]
			err := b.PublishMessage(selectedChannel, message)
			if err != nil {
				log.Printf("Erro ao publicar mensagem: %v", err)

				// Se canal não existe, escolher outro ou criar um novo
				availableChannels, listErr := b.ListChannels()
				if listErr == nil && len(availableChannels) > 0 {
					b.channels = availableChannels
					selectedChannel = b.channels[rand.Intn(len(b.channels))]
				} else if len(b.channels) == 0 {
					// Criar um novo canal se não há nenhum
					channelName := fmt.Sprintf("canal%d", rand.Intn(10000))
					err := b.CreateChannel(channelName)
					if err == nil {
						b.channels = []string{channelName}
						selectedChannel = channelName
					}
				}
				// Continuar mesmo se falhar
			} else {
				b.messageCount++
				fmt.Printf("Bot '%s' enviou mensagem %d no canal '%s': %s\n",
					b.username, b.messageCount, selectedChannel, message)
			}

			// Pausa entre mensagens
			time.Sleep(time.Duration(rand.Intn(3)+1) * time.Second)
		}

		// Pausa antes do próximo ciclo
		fmt.Printf("Bot '%s' pausando por 10 segundos...\n", b.username)
		time.Sleep(10 * time.Second)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	b := newBot()
	defer b.Close()

	err := b.Connect()
	if err != nil {
		log.Fatal("Erro ao conectar:", err)
	}

	b.Run()
}
