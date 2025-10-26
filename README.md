# Sistema de Mensagens Instantâneas Distribuído

Este projeto implementa um sistema completo de mensagens distribuído usando ZeroMQ, seguindo os padrões de sistemas distribuídos estudados na disciplina.

## Arquitetura

### Componentes

- **Broker (Python)**: Proxy ROUTER-DEALER para balanceamento round-robin entre clientes e servidores
- **Proxy (Python)**: Proxy XPUB-XSUB para comunicação Publisher-Subscriber
- **Servidor de Referência (Python)**: Gerencia ranks, heartbeat e lista de servidores ativos
- **Servidores (Node.js)**: 3 réplicas que processam requisições e publicam mensagens
- **Cliente Interativo (Go)**: CLI para usuário real
- **Bots (Go)**: 2 réplicas de clientes automáticos gerando mensagens

### Linguagens Utilizadas

- **Python** (3.13): Broker, Proxy, Servidor de Referência
- **JavaScript/Node.js** (18): Servidores (3 réplicas)
- **Go** (1.21): Clientes (interativo e bots)

## Funcionalidades Implementadas

### Parte 1: Request-Reply ✅

- Login de usuários
- Listagem de usuários cadastrados
- Criação e listagem de canais
- Persistência em arquivos JSON

### Parte 2: Publisher-Subscriber ✅

- Publicação de mensagens em canais
- Mensagens privadas entre usuários
- Bot automático gerando mensagens

### Parte 3: MessagePack ✅

- Serialização binária para todas as mensagens
- Redução do tamanho das mensagens

### Parte 4: Relógios e Sincronização ✅

- Relógio lógico em todos os processos
- Servidor de referência para gerenciamento de ranks
- Sincronização Berkeley entre servidores
- Heartbeat para monitoramento de servidores

### Parte 5: Consistência e Replicação ✅

- Primary-Backup Passivo implementado
- Replicação síncrona entre servidores
- Eleição automática de novo primary em caso de falha

## Como Executar o Projeto

### Pré-requisitos

- Docker instalado
- Docker Compose instalado
- Portas 5555-5562 disponíveis

### Passo a Passo

1. **Clone o repositório** (se ainda não fez):

```bash
git clone <url-do-repositorio>
cd sistemas-distribuidos
```

2. **Navegue até a pasta src**:

```bash
cd src
```

3. **Inicie todos os containers**:

```bash
docker-compose up --build
```

Isso irá:

- Construir todas as imagens Docker necessárias
- Iniciar o broker na porta 5555-5556
- Iniciar o proxy na porta 5557-5558
- Iniciar o servidor de referência na porta 5559
- Iniciar 3 réplicas de servidores (portas 5560-5562)
- Iniciar 1 cliente interativo
- Iniciar 2 bots automáticos

### Verificar Status

Para verificar se todos os containers estão rodando:

```bash
docker ps
```

Você deve ver containers com os nomes:

- `broker`
- `proxy`
- `reference`
- `src-server-1`, `src-server-2`, `src-server-3`
- `src-client-1`
- `src-bot-1`, `src-bot-2`

### Ver os Logs

Para ver logs de um container específico:

```bash
# Ver logs dos bots
docker logs -f src-bot-1
docker logs -f src-bot-2

# Ver logs dos servidores
docker logs -f src-server-1
docker logs -f src-server-2

# Ver logs do broker
docker logs -f broker

# Ver logs do proxy
docker logs -f proxy

# Ver todos os logs
docker-compose logs -f
```

### Usar o Cliente Interativo

1. **Conecte ao container do cliente**:

```bash
docker exec -it src-client-1 sh
```

2. **Use os comandos disponíveis**:

```
login <nome>           - Fazer login
users                  - Listar usuários
channels               - Listar canais
create <canal>         - Criar canal
pub <canal> <mensagem> - Publicar no canal
msg <usuário> <mensagem> - Enviar mensagem privada
quit                   - Sair
```

### Exemplo de Sessão

```bash
# Dentro do container do cliente
login alice
users
channels
create geral
pub geral Olá mundo!
msg bot123 Oi!
```

### Parar o Sistema

Para parar todos os containers:

```bash
docker-compose down
```

Para parar e remover volumes (limpa dados persistidos):

```bash
docker-compose down -v
```

## Estrutura do Projeto

```
.
├── src/
│   ├── broker/           # Broker Python (ROUTER-DEALER)
│   │   └── main.py
│   ├── proxy/            # Proxy Python (XPUB-XSUB)
│   │   └── main.py
│   ├── reference/        # Servidor de Referência Python
│   │   └── main.py
│   ├── server/           # Servidores Node.js (3 réplicas)
│   │   ├── main.js
│   │   ├── main.py
│   │   └── package.json
│   ├── client/           # Cliente Go (interativo e bots)
│   │   ├── bot.go
│   │   ├── main.go
│   │   ├── go.mod
│   │   └── go.sum
│   ├── docker-compose.yml
│   ├── Dockerfile.python
│   ├── Dockerfile.node
│   └── Dockerfile.go
├── enunciados/           # Enunciados das partes
├── README.md            # Este arquivo
└── STATUS.md            # Status atual do projeto
```

## Persistência

Todos os dados são persistidos em arquivos JSON dentro dos servidores:

- `users.json` - Usuários cadastrados
- `channels.json` - Canais criados
- `messages.json` - Mensagens privadas
- `publications.json` - Publicações em canais

## Formato de Mensagens

Todas as mensagens seguem o padrão MessagePack:

```json
{
  "service": "nome_servico",
  "data": {
    "timestamp": 1234567890,
    "clock": 42
    // ... campos específicos
  }
}
```

## Testando Funcionalidades

### 1. Teste de Login

```bash
docker exec -it src-client-1 sh
login alice
users
```

### 2. Teste de Canais

```bash
create geral
channels
```

### 3. Teste de Publicação

```bash
pub geral Minha primeira mensagem!
```

### 4. Observar Bots

Os bots geram mensagens automaticamente. Veja os logs:

```bash
docker logs -f src-bot-1
```

### 5. Teste de Falha (Primary-Backup)

```bash
# Derrubar um servidor
docker stop src-server-1

# Sistema continua funcionando com servers 2 e 3
docker exec -it src-client-1 sh
pub geral Teste após falha
```

## Solução de Problemas

### Containers não iniciam

```bash
# Limpar tudo e reconstruir
docker-compose down -v
docker-compose up --build
```

### Porta já em uso

```bash
# Verificar qual processo está usando a porta
netstat -ano | findstr :5555

# Ou no Linux
lsof -i :5555
```

### Erro de permissão Docker

```bash
# Adicionar usuário ao grupo docker (Linux)
sudo usermod -aG docker $USER
# Re-fazer login
```

## Recursos Adicionais

- Veja `STATUS.md` para status detalhado do projeto
- Veja `enunciados/` para os requisitos de cada parte
- Veja `BOT_FIXES.md` para correções aplicadas nos bots

## Notas

- O sistema usa ZeroMQ v4 (Go) e mensagens MessagePack para serialização
- Todos os processos implementam relógio lógico para ordenação de eventos
- O sistema suporta tolerância a falhas através de replicação Primary-Backup
- Logs de sincronização e heartbeat aparecem nos consoles dos containers
