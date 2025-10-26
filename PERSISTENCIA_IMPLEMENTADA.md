# Persistência em Arquivos JSON Implementada

## Status: ✅ COMPLETA E FUNCIONANDO

## Arquivos JSON Criados

### 1. `users.json`

Armazena todos os logins dos usuários com:

- **user**: Nome do usuário
- **loginTime**: Timestamp do login
- **lastSeen**: Último acesso
- **createdAt**: Data de criação

**Exemplo:**

```json
[
  [
    "BotRho15",
    {
      "user": "BotRho15",
      "loginTime": 1761445670213,
      "lastSeen": 1761445670215,
      "createdAt": 1761445670213
    }
  ]
]
```

### 2. `channels.json`

Armazena todos os canais criados pelos usuários.

**Exemplo:**

```json
["canal614"]
```

### 3. `publications.json`

Armazena todas as publicações em canais com:

- **user**: Usuário que publicou
- **channel**: Canal onde foi publicada
- **message**: Mensagem publicada
- \*\*timestamp: Timestamp da publicação
- **clock**: Valor do relógio lógico

**Exemplo:**

```json
[
  {
    "user": "BotMu312",
    "channel": "canal614",
    "message": "Alguém quer conversar?",
    "timestamp": 1761445674002,
    "clock": 13
  },
  {
    "user": "BotMu312",
    "channel": "canal614",
    "message": "Estou testando o sistema",
    "timestamp": 1761445678012,
    "clock": 21
  }
]
```

### 4. `messages.json`

Armazena todas as mensagens privadas entre usuários com:

- **src**: Usuário remetente
- **dst**: Usuário destinatário
- **message**: Mensagem enviada
- **timestamp**: Timestamp da mensagem
- **clock**: Valor do relógio lógico

## Funcionalidades Implementadas

### Salvamento Automático

- ✅ Salva automaticamente a cada 30 segundos
- ✅ Salva imediatamente após cada operação:
  - Login de usuário
  - Criação de canal
  - Publicação em canal
  - Mensagem privada
  - Replicação de dados

### Carregamento Automático

- ✅ Carrega dados ao iniciar o servidor
- ✅ Se arquivos não existirem, inicia com dados vazios
- ✅ Continua funcionando normalmente

### Formatação

- ✅ Arquivos JSON formatados com indentação (legível)
- ✅ Dados estruturados corretamente
- ✅ Preserva todos os metadados (timestamps, clock lógico)

## Logs de Confirmação

```
Persistência: 1 users, 1 channels, 0 messages, 3 publications
Dados salvos automaticamente
```

## Conformidade com Enunciados

### Parte 1 - Request-Reply ✅

- ✅ Login de usuários persistido com timestamp
- ✅ Canais criados são persistidos

### Parte 2 - Publisher-Subscriber ✅

- ✅ Publicações em canais são persistidas
- ✅ Mensagens privadas são persistidas
- ✅ Histórico completo disponível

### Parte 4 - Relógios ✅

- ✅ Timestamps preservados
- ✅ Relógio lógico (clock) armazenado com cada evento

## Testando a Persistência

```bash
# Ver arquivos JSON
docker exec src-server-1 cat users.json
docker exec src-server-1 cat channels.json
docker exec src-server-1 cat publications.json
docker exec src-server-1 cat messages.json

# Parar containers e reiniciar (dados devem ser preservados)
docker-compose down
docker-compose up -d
```

## Observações

1. **Persistência Cross-Server**: Cada servidor mantém seus próprios arquivos JSON
2. **Replicação**: Dados são replicados entre servidores via Primary-Backup
3. **Recuperação**: Sistema pode ser reiniciado sem perder dados
4. **Formato**: JSON estruturado facilita recuperação histórica

## Conclusão

A persistência está **completamente implementada e funcionando** conforme especificado nos enunciados. Os dados são:

- ✅ Armazenados em disco
- ✅ Formatados em JSON
- ✅ Incluem timestamps
- ✅ Incluem relógio lógico
- ✅ Salvos automaticamente
- ✅ Carregados automaticamente
- ✅ Conformes com todas as partes do projeto
