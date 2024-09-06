import flask
from flask import request
import json

app = flask.Flask(__name__)

@app.route('/neworder-python', methods=['POST'])
def neworder_python_subscriber():
    content = request.json
    print(f'got a new order {content}', flush=True)
    return json.dumps({'success': True}), 200, {'ContentType': 'application/json'}

app.run(host='0.0.0.0', port=4000)