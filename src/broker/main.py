import zmq
import msgpack
import logging

# Configurar logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def main():
    context = zmq.Context()
    
    # Socket para clientes (ROUTER)
    client_socket = context.socket(zmq.ROUTER)
    client_socket.bind("tcp://*:5555")
    
    # Socket para servidores (DEALER)
    server_socket = context.socket(zmq.DEALER)
    server_socket.bind("tcp://*:5556")
    
    logger.info("Broker iniciado - ROUTER em porta 5555, DEALER em porta 5556")
    
    try:
        # Proxy simples - repassa mensagens entre clientes e servidores
        zmq.proxy(client_socket, server_socket)
    except KeyboardInterrupt:
        logger.info("Broker encerrando...")
    finally:
        client_socket.close()
        server_socket.close()
        context.term()

if __name__ == "__main__":
    main()
