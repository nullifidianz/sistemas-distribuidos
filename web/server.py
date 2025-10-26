from flask import Flask, request, jsonify, send_from_directory
from flask_cors import CORS
import zmq
import msgpack
import logging
import os

app = Flask(__name__)
CORS(app)

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class ZeroMQProxy:
    def __init__(self):
        self.context = zmq.Context()
        self.socket = self.context.socket(zmq.REQ)
        self.socket.connect("tcp://broker:5555")
        logger.info("Conectado ao broker ZeroMQ")
    
    def send_request(self, service, data):
        try:
            request = {
                'service': service,
                'data': data
            }
            
            encoded = msgpack.packb(request)
            self.socket.send(encoded)
            
            response_bytes = self.socket.recv()
            response = msgpack.unpackb(response_bytes)
            
            return response.get('data', {})
        except Exception as e:
            logger.error(f"Erro na comunicação ZeroMQ: {e}")
            return {'status': 'erro', 'description': str(e)}

zmq_proxy = ZeroMQProxy()

@app.route('/')
def index():
    return send_from_directory('.', 'index.html')

@app.route('/api', methods=['POST'])
def api():
    try:
        data = request.json
        service = data.get('service')
        request_data = data.get('data', {})
        
        # Adicionar timestamp se não existir
        if 'timestamp' not in request_data:
            request_data['timestamp'] = int(__import__('time').time() * 1000)
        
        # Enviar para o sistema via ZeroMQ
        response = zmq_proxy.send_request(service, request_data)
        
        return jsonify(response)
    except Exception as e:
        logger.error(f"Erro na API: {e}")
        return jsonify({'status': 'erro', 'description': str(e)}), 500

if __name__ == '__main__':
    port = int(os.environ.get('PORT', 3000))
    logger.info(f"Servidor web iniciado na porta {port}")
    app.run(host='0.0.0.0', port=port, debug=True)
