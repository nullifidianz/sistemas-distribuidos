import zmq
import msgpack
import logging
import time
import threading
from collections import defaultdict

# Configurar logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class ReferenceServer:
    def __init__(self):
        self.context = zmq.Context()
        self.rep_socket = self.context.socket(zmq.REP)
        
        # Dados do servidor
        self.servers = {}  # {nome: {'rank': int, 'last_heartbeat': timestamp}}
        self.next_rank = 1
        self.logical_clock = 0
        
        # Thread para limpeza de servidores inativos
        self.cleanup_thread = None
        self.running = False
        
        self.setup_socket()
        self.start_cleanup_thread()
    
    def setup_socket(self):
        # Bind na porta 5559 para comunicação com servidores
        self.rep_socket.bind("tcp://*:5559")
        logger.info("Servidor de referência iniciado na porta 5559")
    
    def increment_clock(self):
        self.logical_clock += 1
        return self.logical_clock
    
    def update_clock(self, received_clock):
        self.logical_clock = max(self.logical_clock, received_clock) + 1
        return self.logical_clock
    
    def start_cleanup_thread(self):
        self.running = True
        self.cleanup_thread = threading.Thread(target=self.cleanup_inactive_servers)
        self.cleanup_thread.daemon = True
        self.cleanup_thread.start()
    
    def cleanup_inactive_servers(self):
        """Remove servidores que não enviaram heartbeat há mais de 30 segundos"""
        while self.running:
            try:
                current_time = time.time()
                inactive_servers = []
                
                for server_name, server_data in self.servers.items():
                    if current_time - server_data['last_heartbeat'] > 30:
                        inactive_servers.append(server_name)
                
                for server_name in inactive_servers:
                    logger.info(f"Removendo servidor inativo: {server_name}")
                    del self.servers[server_name]
                
                time.sleep(10)  # Verificar a cada 10 segundos
            except Exception as e:
                logger.error(f"Erro na limpeza de servidores: {e}")
    
    def handle_request(self, request):
        service = request.get('service')
        data = request.get('data', {})
        
        # Atualizar relógio lógico
        if 'clock' in data:
            self.update_clock(data['clock'])
        
        if service == 'rank':
            return self.handle_rank(data)
        elif service == 'list':
            return self.handle_list(data)
        elif service == 'heartbeat':
            return self.handle_heartbeat(data)
        else:
            return {
                'service': service,
                'data': {
                    'status': 'erro',
                    'timestamp': int(time.time() * 1000),
                    'clock': self.increment_clock(),
                    'description': 'Serviço não encontrado'
                }
            }
    
    def handle_rank(self, data):
        server_name = data.get('user')
        
        if not server_name:
            return {
                'service': 'rank',
                'data': {
                    'status': 'erro',
                    'timestamp': int(time.time() * 1000),
                    'clock': self.increment_clock(),
                    'description': 'Nome do servidor não fornecido'
                }
            }
        
        # Se servidor já existe, retornar rank existente
        if server_name in self.servers:
            rank = self.servers[server_name]['rank']
            logger.info(f"Servidor '{server_name}' solicitou rank - rank existente: {rank}")
        else:
            # Novo servidor - atribuir próximo rank
            rank = self.next_rank
            self.servers[server_name] = {
                'rank': rank,
                'last_heartbeat': time.time()
            }
            self.next_rank += 1
            logger.info(f"Novo servidor '{server_name}' registrado com rank: {rank}")
        
        return {
            'service': 'rank',
            'data': {
                'rank': rank,
                'timestamp': int(time.time() * 1000),
                'clock': self.increment_clock()
            }
        }
    
    def handle_list(self, data):
        # Criar lista de servidores com nome e rank
        server_list = []
        for server_name, server_data in self.servers.items():
            server_list.append({
                'name': server_name,
                'rank': server_data['rank']
            })
        
        # Ordenar por rank
        server_list.sort(key=lambda x: x['rank'])
        
        logger.info(f"Solicitação de lista de servidores - {len(server_list)} servidores ativos")
        
        return {
            'service': 'list',
            'data': {
                'list': server_list,
                'timestamp': int(time.time() * 1000),
                'clock': self.increment_clock()
            }
        }
    
    def handle_heartbeat(self, data):
        server_name = data.get('user')
        
        if not server_name:
            return {
                'service': 'heartbeat',
                'data': {
                    'status': 'erro',
                    'timestamp': int(time.time() * 1000),
                    'clock': self.increment_clock(),
                    'description': 'Nome do servidor não fornecido'
                }
            }
        
        # Atualizar heartbeat do servidor
        if server_name in self.servers:
            self.servers[server_name]['last_heartbeat'] = time.time()
            logger.debug(f"Heartbeat recebido de '{server_name}'")
        else:
            # Servidor não registrado - registrar com próximo rank
            rank = self.next_rank
            self.servers[server_name] = {
                'rank': rank,
                'last_heartbeat': time.time()
            }
            self.next_rank += 1
            logger.info(f"Servidor '{server_name}' registrado via heartbeat com rank: {rank}")
        
        return {
            'service': 'heartbeat',
            'data': {
                'timestamp': int(time.time() * 1000),
                'clock': self.increment_clock()
            }
        }
    
    def run(self):
        logger.info("Servidor de referência iniciado")
        
        try:
            while True:
                # Receber requisição
                request_bytes = self.rep_socket.recv()
                
                try:
                    # Deserializar mensagem
                    request = msgpack.unpackb(request_bytes)
                    
                    # Processar requisição
                    response = self.handle_request(request)
                    
                    # Serializar e enviar resposta
                    response_bytes = msgpack.packb(response)
                    self.rep_socket.send(response_bytes)
                    
                except Exception as e:
                    logger.error(f"Erro ao processar requisição: {e}")
                    error_response = {
                        'service': 'error',
                        'data': {
                            'status': 'erro',
                            'timestamp': int(time.time() * 1000),
                            'clock': self.increment_clock(),
                            'description': 'Erro interno do servidor'
                        }
                    }
                    error_bytes = msgpack.packb(error_response)
                    self.rep_socket.send(error_bytes)
        
        except KeyboardInterrupt:
            logger.info("Servidor de referência encerrando...")
        finally:
            self.running = False
            self.rep_socket.close()
            self.context.term()

def main():
    server = ReferenceServer()
    server.run()

if __name__ == "__main__":
    main()
