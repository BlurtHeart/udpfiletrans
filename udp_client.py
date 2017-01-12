#!/usr/bin/env python
import socket
from SocketServer import ThreadingUDPServer, BaseRequestHandler
from time import ctime
import sys
import json

class FileClient(object):
    def __init__(self, host, port):
        self._host = host
        self._port = port
        self.set_address()
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    def __del__(self):
        try:
            self.sock.close()
        except:
            pass
    @property
    def port(self):
        return self._port
    @port.setter
    def port(self, value):
        self._port = value
        self.set_address()
    @property
    def host(self):
        return self._host
    @host.setter
    def host(self, value):
        self._host = value
        self.set_address()
    def set_address(self):
        self.server_addr = (self._host, self._port)
    def send_data(self, data):
        try:
            self.sock.sendto(data, self.server_addr)
            return True
        except:
            return False
    def recv_data(self, recv_size):
        try:
            received_data = self.sock.recv(recv_size)
            return received_data
        except:
            return None
        
if __name__ == "__main__":
    host = '127.0.0.1'
    port = 54321
    recv_size = 1024
    client = FileClient(host, port)
    header = {"filename":'1.txt', "file_md5":"xxxxxxxxxxxxx", "file_path":"/home/steve/", "file_packets":0}
    data = json.dumps(header)
    client.send_data(data)
    received_data = client.recv_data(recv_size)
    print received_data