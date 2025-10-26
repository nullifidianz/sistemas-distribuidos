//go:build client
// +build client

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
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

type clientMessage struct {
	Service string      `msgpack:"service"`
	Data    interface{} `msgpack:"data"`
}

type chatClient struct {
	reqSocket    *zmq4.Socket
	subSocket    *zmq4.Socket
	context      *zmq4.Context
	logicalClock int
	username     string
}

func newChatClient() *chatClient {
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

	return &chatClient{
		reqSocket:    reqSocket,
		subSocket:    subSocket,
		context:      context,
		logicalClock: 0,
	}
}

func (c *chatClient) Connect() error {
	// Conectar ao broker
	err := c.reqSocket.Connect("tcp://broker:5555")
	if err != nil {
		return fmt.Errorf("erro ao conectar ao broker: %v", err)
	}

	// Conectar ao proxy
	err = c.subSocket.Connect("tcp://proxy:5558")
	if err != nil {
		return fmt.Errorf("erro ao conectar ao proxy: %v", err)
	}

	// Subscrever a todas as mensagens
	err = c.subSocket.SetSubscribe("")
	if err != nil {
		return fmt.Errorf("erro ao configurar subscription: %v", err)
	}

	fmt.Println("Conectado ao sistema de mensagens")
	return nil
}

func (c *chatClient) Close() {
	c.reqSocket.Close()
	c.subSocket.Close()
	c.context.Term()
}

func (c *chatClient) incrementClock() int {
	c.logicalClock++
	return c.logicalClock
}

func (c *chatClient) updateClock(receivedClock int) {
	c.logicalClock = max(c.logicalClock, receivedClock) + 1
}

func (c *chatClient) sendRequest(service string, data interface{}) (interface{}, error) {
	// Incrementar relógio lógico antes de enviar
	clock := c.incrementClock()

	// Adicionar clock aos dados se for um mapa
	if dataMap, ok := data.(map[string]interface{}); ok {
		dataMap["clock"] = clock
		data = dataMap
	}

	request := clientMessage{
		Service: service,
		Data:    data,
	}

	// Serializar mensagem
	encoded, err := msgpack.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar mensagem: %v", err)
	}

	// Enviar requisição
	_, err = c.reqSocket.SendBytes(encoded, 0)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar mensagem: %v", err)
	}

	// Receber resposta
	responseBytes, err := c.reqSocket.RecvBytes(0)
	if err != nil {
		return nil, fmt.Errorf("erro ao receber resposta: %v", err)
	}

	// Deserializar resposta
	var response clientMessage
	err = msgpack.Unmarshal(responseBytes, &response)
	if err != nil {
		return nil, fmt.Errorf("erro ao deserializar resposta: %v", err)
	}

	// Atualizar relógio lógico se recebeu clock
	if responseData, ok := response.Data.(map[string]interface{}); ok {
		if receivedClock, exists := responseData["clock"]; exists {
			if clockVal, ok := receivedClock.(int); ok {
				c.updateClock(clockVal)
			}
		}
	}

	return response.Data, nil
}

func (c *chatClient) Login(username string) error {
	data := map[string]interface{}{
		"user":      username,
		"timestamp": time.Now().UnixMilli(),
	}

	response, err := c.sendRequest("login", data)
	if err != nil {
		return err
	}

	// Converter resposta para LoginResponse
	responseData, ok := response.(map[string]interface{})
	if !ok {
		return fmt.Errorf("resposta inválida")
	}

	status, _ := responseData["status"].(string)
	if status == "erro" {
		description, _ := responseData["description"].(string)
		return fmt.Errorf("erro no login: %s", description)
	}

	c.username = username
	fmt.Printf("Login realizado com sucesso como: %s\n", username)
	return nil
}

func (c *chatClient) ListUsers() ([]string, error) {
	data := map[string]interface{}{
		"timestamp": time.Now().UnixMilli(),
	}

	response, err := c.sendRequest("users", data)
	if err != nil {
		return nil, err
	}

	responseData, ok := response.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("resposta inválida")
	}

	users, _ := responseData["users"].([]interface{})
	var userList []string
	for _, user := range users {
		if username, ok := user.(string); ok {
			userList = append(userList, username)
		}
	}

	return userList, nil
}

func (c *chatClient) CreateChannel(channelName string) error {
	data := map[string]interface{}{
		"channel":   channelName,
		"timestamp": time.Now().UnixMilli(),
	}

	response, err := c.sendRequest("channel", data)
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

	fmt.Printf("Canal '%s' criado com sucesso\n", channelName)
	return nil
}

func (c *chatClient) ListChannels() ([]string, error) {
	data := map[string]interface{}{
		"timestamp": time.Now().UnixMilli(),
	}

	response, err := c.sendRequest("channels", data)
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

func (c *chatClient) PublishMessage(channel, message string) error {
	data := map[string]interface{}{
		"user":      c.username,
		"channel":   channel,
		"message":   message,
		"timestamp": time.Now().UnixMilli(),
	}

	response, err := c.sendRequest("publish", data)
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

	fmt.Printf("Mensagem publicada no canal '%s'\n", channel)
	return nil
}

func (c *chatClient) SendPrivateMessage(destUser, message string) error {
	data := map[string]interface{}{
		"src":       c.username,
		"dst":       destUser,
		"message":   message,
		"timestamp": time.Now().UnixMilli(),
	}

	response, err := c.sendRequest("message", data)
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
		return fmt.Errorf("erro ao enviar mensagem: %s", errorMsg)
	}

	fmt.Printf("Mensagem enviada para '%s'\n", destUser)
	return nil
}

func (c *chatClient) ListenForMessages() {
	go func() {
		for {
			// Receber tópico e mensagem
			_, err := c.subSocket.RecvBytes(0) // tópico
			if err != nil {
				log.Printf("Erro ao receber mensagem: %v", err)
				continue
			}

			messageBytes, err := c.subSocket.RecvBytes(0)
			if err != nil {
				log.Printf("Erro ao receber dados da mensagem: %v", err)
				continue
			}

			// Deserializar mensagem
			var message clientMessage
			err = msgpack.Unmarshal(messageBytes, &message)
			if err != nil {
				log.Printf("Erro ao deserializar mensagem: %v", err)
				continue
			}

			// Atualizar relógio lógico
			if messageData, ok := message.Data.(map[string]interface{}); ok {
				if receivedClock, exists := messageData["clock"]; exists {
					if clockVal, ok := receivedClock.(int); ok {
						c.updateClock(clockVal)
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
					fmt.Printf("[%s] %s: %s\n", channel, user, msg)
				}
			case "private_message":
				if messageData, ok := message.Data.(map[string]interface{}); ok {
					src, _ := messageData["src"].(string)
					dst, _ := messageData["dst"].(string)
					msg, _ := messageData["message"].(string)
					if dst == c.username {
						fmt.Printf("[PRIVADO] %s: %s\n", src, msg)
					}
				}
			}
		}
	}()
}

func main() {
	client := newChatClient()
	defer client.Close()

	err := client.Connect()
	if err != nil {
		log.Fatal("Erro ao conectar:", err)
	}

	// Iniciar escuta de mensagens
	client.ListenForMessages()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("=== Sistema de Mensagens Distribuído ===")
	fmt.Println("Comandos disponíveis:")
	fmt.Println("  login <nome> - Fazer login")
	fmt.Println("  users - Listar usuários")
	fmt.Println("  channels - Listar canais")
	fmt.Println("  create <canal> - Criar canal")
	fmt.Println("  pub <canal> <mensagem> - Publicar no canal")
	fmt.Println("  msg <usuário> <mensagem> - Enviar mensagem privada")
	fmt.Println("  quit - Sair")
	fmt.Println()

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		command := parts[0]

		switch command {
		case "login":
			if len(parts) < 2 {
				fmt.Println("Uso: login <nome>")
				continue
			}
			err := client.Login(parts[1])
			if err != nil {
				fmt.Printf("Erro: %v\n", err)
			}

		case "users":
			users, err := client.ListUsers()
			if err != nil {
				fmt.Printf("Erro: %v\n", err)
			} else {
				fmt.Printf("Usuários: %v\n", users)
			}

		case "channels":
			channels, err := client.ListChannels()
			if err != nil {
				fmt.Printf("Erro: %v\n", err)
			} else {
				fmt.Printf("Canais: %v\n", channels)
			}

		case "create":
			if len(parts) < 2 {
				fmt.Println("Uso: create <canal>")
				continue
			}
			err := client.CreateChannel(parts[1])
			if err != nil {
				fmt.Printf("Erro: %v\n", err)
			}

		case "pub":
			if len(parts) < 3 {
				fmt.Println("Uso: pub <canal> <mensagem>")
				continue
			}
			channel := parts[1]
			message := strings.Join(parts[2:], " ")
			err := client.PublishMessage(channel, message)
			if err != nil {
				fmt.Printf("Erro: %v\n", err)
			}

		case "msg":
			if len(parts) < 3 {
				fmt.Println("Uso: msg <usuário> <mensagem>")
				continue
			}
			destUser := parts[1]
			message := strings.Join(parts[2:], " ")
			err := client.SendPrivateMessage(destUser, message)
			if err != nil {
				fmt.Printf("Erro: %v\n", err)
			}

		case "quit":
			fmt.Println("Saindo...")
			return

		default:
			fmt.Println("Comando não reconhecido")
		}
	}
}
