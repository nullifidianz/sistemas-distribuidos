const zmq = require('zeromq');
const msgpack = require('msgpack5')();
const fs = require('fs').promises;
const path = require('path');

class ChatServer {
    constructor() {
        this.repSocket = zmq.socket('rep');
        this.pubSocket = zmq.socket('pub');
        
        // Socket para comunicação com servidor de referência
        this.refSocket = zmq.socket('req');
        
        // Manter contexto para clean up
        this.context = this.repSocket.context;
        
        // Dados persistentes
        this.users = new Map();
        this.channels = new Set();
        this.messages = [];
        this.publications = [];
        
        // Relógio lógico e físico
        this.logicalClock = 0;
        this.physicalClock = Date.now();
        
        // Sincronização Berkeley
        this.serverName = `server_${Math.random().toString(36).substr(2, 9)}`;
        this.rank = null;
        this.coordinator = null;
        this.messageCount = 0;
        this.syncInterval = 10; // Sincronizar a cada 10 mensagens
        
        // Servidores conhecidos
        this.knownServers = new Map();
        
        // Replicação passiva
        this.isPrimary = false;
        this.backupServers = new Set();
        this.replicationSocket = zmq.socket('rep');
        this.replicationPort = 5560;
        
        this.loadData();
        this.setupSockets();
        this.setupReplication();
        this.registerWithReferenceServer();
        this.startHeartbeat();
        this.startAutoSave(); // Iniciar salvamento automático
    }

    async loadData() {
        try {
            // Carregar usuários
            const usersData = await fs.readFile('users.json', 'utf8');
            const users = JSON.parse(usersData);
            this.users = new Map(users);
        } catch (error) {
            console.log('Arquivo users.json não encontrado, iniciando com dados vazios');
        }

        try {
            // Carregar canais
            const channelsData = await fs.readFile('channels.json', 'utf8');
            const channels = JSON.parse(channelsData);
            this.channels = new Set(channels);
        } catch (error) {
            console.log('Arquivo channels.json não encontrado, iniciando com dados vazios');
        }

        try {
            // Carregar mensagens
            const messagesData = await fs.readFile('messages.json', 'utf8');
            this.messages = JSON.parse(messagesData);
        } catch (error) {
            console.log('Arquivo messages.json não encontrado, iniciando com dados vazios');
        }

        try {
            // Carregar publicações
            const publicationsData = await fs.readFile('publications.json', 'utf8');
            this.publications = JSON.parse(publicationsData);
        } catch (error) {
            console.log('Arquivo publications.json não encontrado, iniciando com dados vazios');
        }
    }

    async saveData() {
        try {
            // Salvar usuários com formatação legível
            const usersArray = Array.from(this.users.entries());
            await fs.writeFile('users.json', JSON.stringify(usersArray, null, 2));
            
            // Salvar canais com formatação legível
            const channelsArray = Array.from(this.channels);
            await fs.writeFile('channels.json', JSON.stringify(channelsArray, null, 2));
            
            // Salvar mensagens com formatação legível
            await fs.writeFile('messages.json', JSON.stringify(this.messages, null, 2));
            
            // Salvar publicações com formatação legível
            await fs.writeFile('publications.json', JSON.stringify(this.publications, null, 2));
            
            console.log(`Persistência: ${this.users.size} users, ${this.channels.size} channels, ${this.messages.length} messages, ${this.publications.length} publications`);
        } catch (error) {
            console.error('Erro ao salvar dados:', error);
        }
    }

    setupSockets() {
        // Conectar ao broker (REP)
        this.repSocket.connect('tcp://broker:5556');
        
        // Conectar ao proxy (PUB)
        this.pubSocket.connect('tcp://proxy:5557');
        
        // Conectar ao servidor de referência
        this.refSocket.connect('tcp://reference:5559');

        // Processar requisições
        this.repSocket.on('message', async (data) => {
            try {
                const message = msgpack.decode(data);
                const response = await this.handleRequest(message);
                this.repSocket.send(msgpack.encode(response));
                
                // Incrementar contador de mensagens
                this.messageCount++;
                
                // Sincronizar relógio físico se necessário
                if (this.messageCount % this.syncInterval === 0) {
                    this.syncPhysicalClock();
                }
            } catch (error) {
                console.error('Erro ao processar mensagem:', error);
                const errorResponse = {
                    service: 'error',
                    data: {
                        status: 'erro',
                        timestamp: Date.now(),
                        clock: this.incrementClock(),
                        description: 'Erro interno do servidor'
                    }
                };
                this.repSocket.send(msgpack.encode(errorResponse));
            }
        });

        console.log(`Servidor '${this.serverName}' iniciado e conectado ao broker, proxy e servidor de referência`);
    }
    
    setupReplication() {
        // Bind na porta de replicação
        this.replicationSocket.bind(`tcp://*:${this.replicationPort}`);
        
        this.replicationSocket.on('message', async (data) => {
            try {
                const message = msgpack.decode(data);
                const response = await this.handleReplicationRequest(message);
                this.replicationSocket.send(msgpack.encode(response));
            } catch (error) {
                console.error('Erro ao processar requisição de replicação:', error);
                const errorResponse = {
                    service: 'error',
                    data: {
                        status: 'erro',
                        timestamp: Date.now(),
                        clock: this.incrementClock(),
                        description: 'Erro na replicação'
                    }
                };
                this.replicationSocket.send(msgpack.encode(errorResponse));
            }
        });
        
        console.log(`Socket de replicação configurado na porta ${this.replicationPort}`);
    }

    incrementClock() {
        this.logicalClock++;
        return this.logicalClock;
    }

    updateClock(receivedClock) {
        this.logicalClock = Math.max(this.logicalClock, receivedClock) + 1;
        return this.logicalClock;
    }

    async registerWithReferenceServer() {
        try {
            await new Promise((resolve, reject) => {
                // Solicitar rank
                const rankRequest = {
                    service: 'rank',
                    data: {
                        user: this.serverName,
                        timestamp: Date.now(),
                        clock: this.incrementClock()
                    }
                };
                
                // Enviar requisição
                this.refSocket.send(msgpack.encode(rankRequest));
                
                // Listener para receber resposta
                const messageHandler = (data) => {
                    try {
                        const rankResponse = msgpack.decode(data);
                        
                        if (rankResponse.service === 'rank' && rankResponse.data.rank) {
                            this.rank = rankResponse.data.rank;
                            console.log(`Servidor '${this.serverName}' recebeu rank: ${this.rank}`);
                        }
                        
                        // Remover listener temporário
                        this.refSocket.removeListener('message', messageHandler);
                        resolve();
                    } catch (error) {
                        this.refSocket.removeListener('message', messageHandler);
                        reject(error);
                    }
                };
                
                this.refSocket.on('message', messageHandler);
            });
            
            // Solicitar lista de servidores
            await this.updateServerList();
            
        } catch (error) {
            console.error('Erro ao registrar com servidor de referência:', error);
        }
    }
    
    async updateServerList() {
        try {
            const listResponse = await new Promise((resolve, reject) => {
                const listRequest = {
                    service: 'list',
                    data: {
                        timestamp: Date.now(),
                        clock: this.incrementClock()
                    }
                };
                
                this.refSocket.send(msgpack.encode(listRequest));
                
                // Listener para receber resposta
                const messageHandler = (data) => {
                    try {
                        const response = msgpack.decode(data);
                        this.refSocket.removeListener('message', messageHandler);
                        resolve(response);
                    } catch (error) {
                        this.refSocket.removeListener('message', messageHandler);
                        reject(error);
                    }
                };
                
                this.refSocket.on('message', messageHandler);
            });
            
            if (listResponse.service === 'list' && listResponse.data.list) {
                this.knownServers.clear();
                listResponse.data.list.forEach(server => {
                    this.knownServers.set(server.name, server.rank);
                });
                
                // Determinar coordenador (servidor com menor rank)
                const minRank = Math.min(...Array.from(this.knownServers.values()));
                this.coordinator = Array.from(this.knownServers.entries())
                    .find(([name, rank]) => rank === minRank)[0];
                
                // Determinar se somos o primary (coordenador)
                this.isPrimary = (this.coordinator === this.serverName);
                
                // Atualizar lista de backups (todos os outros servidores)
                this.backupServers.clear();
                this.knownServers.forEach((rank, name) => {
                    if (name !== this.serverName) {
                        this.backupServers.add(name);
                    }
                });
                
                console.log(`Lista de servidores atualizada. Coordenador: ${this.coordinator}, Primary: ${this.isPrimary}`);
            }
        } catch (error) {
            console.error('Erro ao atualizar lista de servidores:', error);
        }
    }
    
    startHeartbeat() {
        setInterval(async () => {
            try {
                await new Promise((resolve, reject) => {
                    const heartbeatRequest = {
                        service: 'heartbeat',
                        data: {
                            user: this.serverName,
                            timestamp: Date.now(),
                            clock: this.incrementClock()
                        }
                    };
                    
                    this.refSocket.send(msgpack.encode(heartbeatRequest));
                    
                    const messageHandler = (data) => {
                        this.refSocket.removeListener('message', messageHandler);
                        resolve();
                    };
                    
                    this.refSocket.on('message', messageHandler);
                });
                
                // Atualizar lista de servidores periodicamente
                await this.updateServerList();
                
            } catch (error) {
                console.error('Erro no heartbeat:', error);
            }
        }, 15000); // Heartbeat a cada 15 segundos
    }
    
    startAutoSave() {
        // Salvar dados automaticamente a cada 30 segundos
        setInterval(async () => {
            try {
                await this.saveData();
                console.log('Dados salvos automaticamente');
            } catch (error) {
                console.error('Erro ao salvar dados automaticamente:', error);
            }
        }, 30000); // A cada 30 segundos
    }
    
    async syncPhysicalClock() {
        if (!this.coordinator || this.coordinator === this.serverName) {
            return; // Não sincronizar se somos o coordenador
        }
        
        try {
            console.log(`Sincronizando relógio físico com coordenador: ${this.coordinator}`);
            // Implementação simplificada - apenas atualizar com timestamp atual
            this.physicalClock = Date.now();
        } catch (error) {
            console.error('Erro na sincronização do relógio físico:', error);
        }
    }

    async handleRequest(request) {
        const { service, data } = request;
        
        // Atualizar relógio lógico
        if (data.clock) {
            this.updateClock(data.clock);
        }

        switch (service) {
            case 'login':
                return await this.handleLogin(data);
            case 'users':
                return await this.handleUsers(data);
            case 'channel':
                return await this.handleChannel(data);
            case 'channels':
                return await this.handleChannels(data);
            case 'publish':
                return await this.handlePublish(data);
            case 'message':
                return await this.handleMessage(data);
            case 'clock':
                return await this.handleClock(data);
            case 'election':
                return await this.handleElection(data);
            default:
                return {
                    service: service,
                    data: {
                        status: 'erro',
                        timestamp: Date.now(),
                        clock: this.incrementClock(),
                        description: 'Serviço não encontrado'
                    }
                };
        }
    }

    async handleLogin(data) {
        const { user, timestamp } = data;
        
        if (!user || user.trim() === '') {
            return {
                service: 'login',
                data: {
                    status: 'erro',
                    timestamp: Date.now(),
                    clock: this.incrementClock(),
                    description: 'Nome de usuário inválido'
                }
            };
        }

        // Registrar usuário com timestamp completo
        this.users.set(user, {
            user: user,
            loginTime: timestamp,
            lastSeen: Date.now(),
            createdAt: timestamp
        });

        // Se somos primary, replicar para backups
        if (this.isPrimary) {
            await this.replicateToBackups('login', {
                user: user,
                timestamp: timestamp,
                clock: this.incrementClock()
            });
        }

        await this.saveData();

        return {
            service: 'login',
            data: {
                status: 'sucesso',
                timestamp: Date.now(),
                clock: this.incrementClock()
            }
        };
    }

    async handleUsers(data) {
        const userList = Array.from(this.users.keys());
        
        return {
            service: 'users',
            data: {
                timestamp: Date.now(),
                clock: this.incrementClock(),
                users: userList
            }
        };
    }

    async handleChannel(data) {
        const { channel, timestamp } = data;
        
        if (!channel || channel.trim() === '') {
            return {
                service: 'channel',
                data: {
                    status: 'erro',
                    timestamp: Date.now(),
                    clock: this.incrementClock(),
                    description: 'Nome do canal inválido'
                }
            };
        }

        if (this.channels.has(channel)) {
            return {
                service: 'channel',
                data: {
                    status: 'erro',
                    timestamp: Date.now(),
                    clock: this.incrementClock(),
                    description: 'Canal já existe'
                }
            };
        }

        // Criar canal
        this.channels.add(channel);
        
        // Se somos primary, replicar para backups
        if (this.isPrimary) {
            await this.replicateToBackups('channel', {
                channel: channel,
                timestamp: timestamp,
                clock: this.incrementClock()
            });
        }
        
        await this.saveData();

        return {
            service: 'channel',
            data: {
                status: 'sucesso',
                timestamp: Date.now(),
                clock: this.incrementClock()
            }
        };
    }

    async handleChannels(data) {
        const channelList = Array.from(this.channels);
        
        return {
            service: 'channels',
            data: {
                timestamp: Date.now(),
                clock: this.incrementClock(),
                channels: channelList
            }
        };
    }

    async handlePublish(data) {
        const { user, channel, message, timestamp } = data;
        
        if (!this.channels.has(channel)) {
            return {
                service: 'publish',
                data: {
                    status: 'erro',
                    timestamp: Date.now(),
                    clock: this.incrementClock(),
                    message: 'Canal não existe'
                }
            };
        }

        // Armazenar publicação
        const publication = {
            user,
            channel,
            message,
            timestamp,
            clock: this.incrementClock()
        };
        
        this.publications.push(publication);
        
        // Se somos primary, replicar para backups
        if (this.isPrimary) {
            await this.replicateToBackups('publish', {
                user,
                channel,
                message,
                timestamp,
                clock: this.incrementClock()
            });
        }
        
        await this.saveData();

        // Publicar no canal
        const pubMessage = {
            service: 'publication',
            data: {
                user,
                channel,
                message,
                timestamp,
                clock: this.incrementClock()
            }
        };

        this.pubSocket.send([channel, msgpack.encode(pubMessage)]);

        return {
            service: 'publish',
            data: {
                status: 'OK',
                timestamp: Date.now(),
                clock: this.incrementClock()
            }
        };
    }

    async handleMessage(data) {
        const { src, dst, message, timestamp } = data;
        
        if (!this.users.has(dst)) {
            return {
                service: 'message',
                data: {
                    status: 'erro',
                    timestamp: Date.now(),
                    clock: this.incrementClock(),
                    message: 'Usuário de destino não existe'
                }
            };
        }

        // Armazenar mensagem
        const msg = {
            src,
            dst,
            message,
            timestamp,
            clock: this.incrementClock()
        };
        
        this.messages.push(msg);
        
        // Se somos primary, replicar para backups
        if (this.isPrimary) {
            await this.replicateToBackups('message', {
                src,
                dst,
                message,
                timestamp,
                clock: this.incrementClock()
            });
        }
        
        await this.saveData();

        // Publicar para o usuário de destino
        const pubMessage = {
            service: 'private_message',
            data: {
                src,
                dst,
                message,
                timestamp,
                clock: this.incrementClock()
            }
        };

        this.pubSocket.send([dst, msgpack.encode(pubMessage)]);

        return {
            service: 'message',
            data: {
                status: 'OK',
                timestamp: Date.now(),
                clock: this.incrementClock()
            }
        };
    }
    
    async handleClock(data) {
        return {
            service: 'clock',
            data: {
                time: this.physicalClock,
                timestamp: Date.now(),
                clock: this.incrementClock()
            }
        };
    }
    
    async handleElection(data) {
        // Resposta simples para eleição
        return {
            service: 'election',
            data: {
                election: 'OK',
                timestamp: Date.now(),
                clock: this.incrementClock()
            }
        };
    }
    
    async replicateToBackups(service, data) {
        if (this.backupServers.size === 0) {
            return; // Não há backups para replicar
        }
        
        const replicationRequest = {
            service: `replicate_${service}`,
            data: {
                ...data,
                timestamp: Date.now(),
                clock: this.incrementClock()
            }
        };
        
        const replicationPromises = Array.from(this.backupServers).map(async (backupName) => {
            try {
                const backupSocket = this.context.socket('req');
                backupSocket.connect(`tcp://${backupName}:${this.replicationPort}`);
                
                backupSocket.send(msgpack.encode(replicationRequest));
                
                const response = await new Promise((resolve, reject) => {
                    const messageHandler = (data) => {
                        try {
                            const decoded = msgpack.decode(data);
                            backupSocket.removeListener('message', messageHandler);
                            resolve(decoded);
                        } catch (error) {
                            backupSocket.removeListener('message', messageHandler);
                            reject(error);
                        }
                    };
                    backupSocket.on('message', messageHandler);
                });
                
                backupSocket.close();
                
                if (response.service === `replicate_${service}` && response.data.status === 'OK') {
                    console.log(`Replicação para ${backupName} bem-sucedida`);
                    return true;
                } else {
                    console.log(`Replicação para ${backupName} falhou`);
                    return false;
                }
            } catch (error) {
                console.error(`Erro na replicação para ${backupName}:`, error);
                return false;
            }
        });
        
        const results = await Promise.all(replicationPromises);
        const successCount = results.filter(r => r).length;
        
        console.log(`Replicação concluída: ${successCount}/${this.backupServers.size} backups confirmaram`);
    }
    
    async handleReplicationRequest(request) {
        const { service, data } = request;
        
        // Atualizar relógio lógico
        if (data.clock) {
            this.updateClock(data.clock);
        }
        
        switch (service) {
            case 'replicate_login':
                return await this.handleReplicateLogin(data);
            case 'replicate_channel':
                return await this.handleReplicateChannel(data);
            case 'replicate_publish':
                return await this.handleReplicatePublish(data);
            case 'replicate_message':
                return await this.handleReplicateMessage(data);
            default:
                return {
                    service: service,
                    data: {
                        status: 'erro',
                        timestamp: Date.now(),
                        clock: this.incrementClock(),
                        description: 'Serviço de replicação não encontrado'
                    }
                };
        }
    }
    
    async handleReplicateLogin(data) {
        const { user, timestamp } = data;
        
        this.users.set(user, {
            loginTime: timestamp,
            lastSeen: Date.now()
        });
        
        await this.saveData();
        
        return {
            service: 'replicate_login',
            data: {
                status: 'OK',
                timestamp: Date.now(),
                clock: this.incrementClock()
            }
        };
    }
    
    async handleReplicateChannel(data) {
        const { channel } = data;
        
        this.channels.add(channel);
        await this.saveData();
        
        return {
            service: 'replicate_channel',
            data: {
                status: 'OK',
                timestamp: Date.now(),
                clock: this.incrementClock()
            }
        };
    }
    
    async handleReplicatePublish(data) {
        const { user, channel, message, timestamp } = data;
        
        const publication = {
            user,
            channel,
            message,
            timestamp,
            clock: this.incrementClock()
        };
        
        this.publications.push(publication);
        await this.saveData();
        
        return {
            service: 'replicate_publish',
            data: {
                status: 'OK',
                timestamp: Date.now(),
                clock: this.incrementClock()
            }
        };
    }
    
    async handleReplicateMessage(data) {
        const { src, dst, message, timestamp } = data;
        
        const msg = {
            src,
            dst,
            message,
            timestamp,
            clock: this.incrementClock()
        };
        
        this.messages.push(msg);
        await this.saveData();
        
        return {
            service: 'replicate_message',
            data: {
                status: 'OK',
                timestamp: Date.now(),
                clock: this.incrementClock()
            }
        };
    }
}

// Iniciar servidor
const server = new ChatServer();

// Graceful shutdown
process.on('SIGINT', async () => {
    console.log('Encerrando servidor...');
    await server.saveData();
    process.exit(0);
});
