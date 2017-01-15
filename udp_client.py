#!/usr/bin/env python
import socket
from time import ctime
import sys
import os
import json
import copy
import hashlib

def calculate_md5(data):
    m = hashlib.md5()
    m.update(data)
    return m.hexdigest()

def calculate_file_md5(fp):
    m = hashlib.md5()
    while True:
        data = fp.read(1024)
        if not data:
            break
        m.update(data)
    return m.hexdigest()

class UdpClient(object):
    def __init__(self, host, port):
        self._host = host
        self._port = port
        self.set_address()
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.sock.settimeout(5)

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
            received_data, client_addr = self.sock.recvfrom(recv_size)
            return (received_data, client_addr)
        except socket.timeout:
            return (None, None)

class FileClient(object):
    def __init__(self, host, port, cfg):
        self.udpclient = UdpClient(host, port)
        self.block_size = 1024
        self.recv_size = 1500
        self.send_path = cfg['send_path']
        self.recv_path = cfg['recv_path']
        self.filename = cfg['filename']
        self.real_file = os.path.join(self.send_path, self.filename)
        self.fp = open(self.real_file, 'r')
        self.calculate_file()
        self.header = self.create_header()

        self.try_times = 5

    def __del__(self):
        if self.fp:
            self.fp.close()

    def create_header(self):
        header = {}
        header['filename'] = self.filename
        header['file_path'] = self.recv_path
        header['file_md5'] = self.file_md5
        header['file_packets'] = self.file_packets
        header['status'] = 'syn'
        return header

    def calculate_file(self):
        try:
            self.file_md5 = calculate_file_md5(self.fp)
            self.file_size = os.path.getsize(self.real_file)
            # round up to an integer; ceil
            self.file_packets = (self.file_size + self.block_size - 1) / self.block_size
        except Exception as exp:
            print 'error:', exp

    def send_data(self, data):
        jsn_data = json.dumps(data)
        print 'send_data:', data
        self.udpclient.send_data(jsn_data)

    def recv_data(self, try_times):
        i = 0
        while i < try_times:
            recv_dict, server_addr = self.udpclient.recv_data(self.recv_size)
            print 'recv data:', recv_dict
            if recv_dict is not None:
                break
        if i == try_times:
            return (None, None)
        else:
            return (json.loads(recv_dict), server_addr)

    def shakehands(self):
        self.send_data(self.header)
        recv_data, server_addr = self.recv_data(self.try_times)
        if recv_data is not None and recv_data['filename'] == self.filename: 
            if recv_data['status'] == 'syn-ack':
                self.udpclient.host = server_addr[0]
                self.udpclient.port = server_addr[1]
                return 1
            elif recv_data['status'] == 'exist':
                return 2
        else:
            return 0

    def send_file(self):
        shake_result = self.shakehands()
        if shake_result == 0:
            return False
        elif shake_result == 2:
            return True
        packet_index = 0
        i_times = 0
        send_finish = False

        while i_times < self.try_times:
            file_offset = packet_index * self.block_size
            if file_offset >= self.file_size:
                server_data, server_addr = self.recv_data(self.try_times)
                if server_data is not None and server_data['status'] == 'finished' and server_data['filename'] == self.filename:
                    send_finish = True
                break
            self.fp.seek(file_offset)
            body = self.fp.read(self.block_size)
            packet_md5 = calculate_md5(body)
            dict_data = copy.deepcopy(self.header)
            dict_data['status'] = 'block'
            dict_data['body'] = body
            dict_data['packet_md5'] = packet_md5
            dict_data['file_offset'] = file_offset
            dict_data['packet_index'] = packet_index
            self.send_data(dict_data)
            server_data, server_addr = self.recv_data(self.try_times)
            if server_data is not None and server_data['status'] == 'block-ack' and server_data['packet_index'] == packet_index and server_data['filename'] == self.filename:
                packet_index += 1
                i_times = 0
            else:
                i_times += 1
        return send_finish

    def run(self):
        send_result = self.send_file()
        print self.real_file, send_result
   
if __name__ == "__main__":
    host = '127.0.0.1'
    port = 54321
    recv_size = 1024
    
    header = {"filename":'1.txt', "file_md5":"xxxxxxxxxxxxx", "send_path":"/home/steve/", "file_packets":1, "recv_path":"/home/steve/workspace/"}
    client = FileClient(host, port, header)
    result = client.run()