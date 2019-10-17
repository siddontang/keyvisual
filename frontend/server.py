from flask import Flask, request
from flask_cors import CORS

from clustergrammer import Network


def genNetworkFromMatrix(matr):
    net = Network()
    # net.load_file('txt/example.txt')
    net.load_file_as_string(matr)
    net.make_clust(run_clustering=False, dendro=False, views=[])
    return net.export_net_json('viz', 'no-indent')


app = Flask(__name__)
cors = CORS(app, headers='Content-Type', resources={r"/*": {"origins": "*"}})
app.config['CORS_HEADERS'] = 'Content-Type'


@app.route('/convert', methods=['GET', 'OPTIONS', 'POST'])
def gen_network():
    content = request.json
    print(content['data'])
    json = genNetworkFromMatrix(content['data'])
    print(json)
    return json


@app.route('/')
def hello_world():
    return 'Hello, World!'


if __name__ == '__main__':
    app.run()
