#!/usr/bin/env python
import socket
from time import ctime
import sys
import os
import msgpack
import copy
import hashlib
import threading
from proto import ProtoCode

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
        self.sock.settimeout(3)

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

        self.recv_times = 3
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
        header['status'] = ProtoCode.SYN
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
        jsn_data = msgpack.packb(data)
        self.udpclient.send_data(jsn_data)

    def recv_data(self, recv_times):
        i = 0
        while i < recv_times:
            recv_dict, server_addr = self.udpclient.recv_data(self.recv_size)
            if recv_dict is not None:
                break
            i += 1
        if i == recv_times:
            return (None, None)
        else:
            return (msgpack.unpackb(recv_dict), server_addr)

    def shakehands(self):
        self.send_data(self.header)
        recv_data, server_addr = self.recv_data(self.recv_times)
        if recv_data is not None and recv_data['filename'] == self.filename: 
            if recv_data['status'] == ProtoCode.ACK:
                self.udpclient.host = server_addr[0]
                self.udpclient.port = server_addr[1]
                return 1
            elif recv_data['status'] == ProtoCode.EXIST:
                return 2
        else:
            return 0

    def send_thread(self):
        file_index = 0
        while file_index < self.file_packets:
            file_offset = file_index * self.block_size
            self.fp.seek(file_offset)
            body = self.fp.read(self.block_size)
            packet_md5 = calculate_md5(body)
            dict_data = copy.deepcopy(self.header)
            dict_data['status'] = ProtoCode.BLOCK
            dict_data['body'] = body
            dict_data['packet_md5'] = packet_md5
            dict_data['file_offset'] = file_offset
            dict_data['packet_index'] = file_index
            self.send_data(dict_data)
            file_index += 1

    def recv_thread(self):
        while True:
            server_data, server_addr = self.recv_data(self.recv_times)
            if server_data is not None and server_data['filename'] == self.filename:
                if server_data['status'] == ProtoCode.COMPLETE:
                    self.send_result = True
                    break
                elif server_data['status'] == ProtoCode.BLOCKACK:
                    pass
                elif server_data['status'] == ProtoCode.BLOCKUNCORRECT:
                    file_offset = server_data['packet_index'] * self.block_size
                    self.fp.seek(file_offset)
                    body = self.fp.read(self.block_size)
                    packet_md5 = calculate_md5(body)
                    dict_data = copy.deepcopy(self.header)
                    dict_data['packet_index'] = server_data['packet_index']
                    dict_data['packet_md5'] = packet_md5
                    dict_data['file_offset'] = file_offset
                    self.send_data(dict_data)
                elif server_data['status'] == ProtoCode.FAILED:
                    self.send_result = False
                    break
            else:
                self.send_result = False
                break

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
                server_data, server_addr = self.recv_data(self.recv_times)
                if server_data is not None and server_data['status'] == ProtoCode.COMPLETE and server_data['filename'] == self.filename:
                    send_finish = True
                break
            self.fp.seek(file_offset)
            body = self.fp.read(self.block_size)
            packet_md5 = calculate_md5(body)
            dict_data = copy.deepcopy(self.header)
            dict_data['status'] = ProtoCode.BLOCK
            dict_data['body'] = body
            dict_data['packet_md5'] = packet_md5
            dict_data['file_offset'] = file_offset
            dict_data['packet_index'] = packet_index
            self.send_data(dict_data)
            server_data, server_addr = self.recv_data(self.recv_times)
            if server_data is not None and server_data['status'] == ProtoCode.BLOCKACK and server_data['packet_index'] == packet_index and server_data['filename'] == self.filename:
                packet_index += 1
                i_times = 0
            else:
                i_times += 1
        return send_finish

    def run(self):
        # send_result = self.send_file()
        # print self.real_file, send_result
        shake_result = self.shakehands()
        if shake_result == 0:
            self.send_result = False
        elif shake_result == 2:
            self.send_result = True
        else:
            ts = threading.Thread(target=self.send_thread, args=(), name='send')
            tr = threading.Thread(target=self.recv_thread, args=(), name='recv')
            tr.start()
            ts.start()
            ts.join()
            tr.join()
        print self.real_file, self.send_result
   
if __name__ == "__main__":
    host = '127.0.0.1'
    port = 54321
    recv_size = 1024
    
    header = {"filename":'1.txt',  "send_path":"/Users/steve/", "recv_path":"/Users/steve/workspace/"}
    client = FileClient(host, port, header)
    result = client.run()