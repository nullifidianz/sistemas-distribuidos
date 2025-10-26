# Como Usar a Interface Web

## Iniciando o Sistema

```bash
cd src
docker compose up --build
```

## Acessando a Interface Web

Abra seu navegador e acesse: **http://localhost:3000**

## Funcionalidades da Interface

### 1. Login

- Digite seu nome de usuário no campo "Nome de Usuário"
- Clique em "Login"

### 2. Listar Usuários e Canais

- Clique em "Listar Usuários" para ver todos os usuários online
- Clique em "Listar Canais" para ver todos os canais disponíveis

### 3. Criar Canal

- Digite um nome para o canal
- Clique em "Criar Canal"

### 4. Publicar Mensagem

- Digite o nome do canal no campo "Canal para Publicar"
- Digite sua mensagem
- Clique em "Publicar no Canal"

### 5. Enviar Mensagem Privada

- Digite o nome do usuário destino no campo "Usuário Destino"
- Digite sua mensagem
- Clique em "Enviar Mensagem Privada"

## Monitoramento

A interface atualiza automaticamente a cada 10 segundos:

- Usuários online
- Canais disponíveis
- Status da conexão

## Logs em Tempo Real

O painel de logs mostra todas as ações realizadas e suas respostas do sistema.
