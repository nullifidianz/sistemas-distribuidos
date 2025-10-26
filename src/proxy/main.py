import zmq
import logging

# Configurar logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def main():
    context = zmq.Context()
    
    # Socket para publishers (XPUB)
    pub_socket = context.socket(zmq.XPUB)
    pub_socket.bind("tcp://*:5558")
    
    # Socket para subscribers (XSUB)
    sub_socket = context.socket(zmq.XSUB)
    sub_socket.bind("tcp://*:5557")
    
    logger.info("Proxy iniciado - XPUB em porta 5558, XSUB em porta 5557")
    
    try:
        # Proxy simples - repassa mensagens entre publishers e subscribers
        zmq.proxy(pub_socket, sub_socket)
    except KeyboardInterrupt:
        logger.info("Proxy encerrando...")
    finally:
        pub_socket.close()
        sub_socket.close()
        context.term()

if __name__ == "__main__":
    main()
