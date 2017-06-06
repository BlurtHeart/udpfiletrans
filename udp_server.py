#!/usr/bin/env python
import socket
from SocketServer import ThreadingUDPServer, BaseRequestHandler
import msgpack
import os
import hashlib
import time
from proto import ProtoCode

host = '127.0.0.1'
port = 54321
addr = (host, port)

class CheckResult:
    UNKNOWN = 0
    NORMAL = 1
    EXIST = 2
    BROKEN = 3

def calculate_md5(data):
    m = hashlib.md5()
    m.update(data)
    return m.hexdigest()

def calculate_file_md5(filename):
    m = hashlib.md5()
    with open(filename, 'r') as fp:
        while True:
            data = fp.read(1024)
            if not data:
                break
            m.update(data)
    return m.hexdigest()


def timedec(func):
    def wrapper(*args, **kwargs):
        start = time.time()
        func(*args, **kwargs)
        end = time.time()
        print 'cost %s seconds' % (end-start)
    return wrapper


class MyRequestHandler(BaseRequestHandler):
    @timedec
    def handle(self):
        print 'got connection from', self.client_address
        self.socket = self.request[1]
        self.data = self.request[0]
        self.recv_size = 1500   # the capacity of the udp packet 
        self.header = msgpack.unpackb(self.data)
        
        handle_result = self.handle_header()
        if handle_result == CheckResult.NORMAL:
            self.handle_body()

    def __del__(self):
        if hasattr(self, 'fp'):
            self.fp.close()

    def handle_header(self):

        if self.header['status'] != ProtoCode.SYN:
            return CheckResult.UNKNOWN

        self.create_new_socket()  
        self.filename = self.header['filename']
        self.file_md5 = self.header['file_md5']
        self.file_path = self.header['file_path']
        self.full_packets = self.header['file_packets']
        self.received_packets_list = []
        self.real_file = os.path.join(self.file_path, self.filename)
        if not os.path.isdir(self.file_path):
            os.makedirs(self.file_path)
        if not os.path.isfile(self.real_file):
            #os.mknod(self.real_file)  # mknod must execute with super-user privileges   
            with open(self.real_file, 'w') as fp:
                pass
        self.fp = open(self.real_file, 'r+')
        self.max_try_times = 5

        data = {}
        data['filename'] = self.filename
        if os.path.isfile(self.real_file):
            check_result = self.check_file_md5()
            if check_result == True:
                data['status'] = ProtoCode.EXIST
                self.send_data(data)
                return CheckResult.EXIST
            else:
                self.fp.truncate()          # or resume broken transfer
                data['status'] = ProtoCode.ACK     # here file not complete
                self.send_data(data)
                return CheckResult.NORMAL
        else:
            data['status'] = ProtoCode.ACK
            self.send_data(data)
            return CheckResult.NORMAL

    def create_new_socket(self):
        self.new_socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.new_socket.settimeout(3)
        # new socket will form the client through the first ack packet
        # here's the question: Is the server's addr same with the addr which client received
        # and will send to?
    def handle_body(self):
        try_times = 0
        recv_finish = False
        while True:
            if self.check_packets():
                recv_finish = True
                break
            try:
                recv_data = self.new_socket.recv(self.recv_size)
                recv_dict = msgpack.unpackb(recv_data)
                self.handle_file_data(recv_dict)
                return_dict = dict()
                return_dict['status'] = ProtoCode.BLOCKACK
                return_dict['filename'] = self.filename
                return_dict['packet_index'] = recv_dict['packet_index']
                self.send_data(return_dict)
                try_times = 0
            except socket.timeout:
                try_times += 1
                if try_times > self.max_try_times:
                    break
        data = {}
        data['filename'] = self.filename
        if recv_finish is True:
            check_result = self.check_file_md5()                    
            if check_result is True:
                data['status'] = ProtoCode.COMPLETE
                print self.filename, 'received True'
            else:
                data['status'] = ProtoCode.FAILED
        else:
            data['status'] = ProtoCode.FAILED    
        self.send_data(data)

    def check_file_md5(self):
        # check self.file_md5 and md5(self.filename)
        file_md5 = calculate_file_md5(self.real_file)
        if file_md5 == self.file_md5:
            return True
        else:
            return False

    def handle_file_data(self, recv_dict):
        # handle packet which include part of filedata and information of the part
        packet_md5 = calculate_md5(recv_dict['body'])
        if packet_md5 != recv_dict['packet_md5']:
            raise socket.timeout    # raise is important

        if isinstance(recv_dict['packet_index'], int):
            if recv_dict['packet_index'] in self.received_packets_list:
                raise socket.timeout    # define another error type
            else:
                self.received_packets_list.append(recv_dict['packet_index'])
        self.write_file(recv_dict['file_offset'], recv_dict['body'])

    def write_file(self, packet_offset, file_data):
        # fseek to packet_offset position, and write file_data into self.filename
        self.fp.seek(packet_offset)
        self.fp.write(file_data)

    def check_packets(self):
        # if all packets which formed the whole file are received, then return true; 
        # on the contrary, return false
        for i in xrange(0, self.full_packets):
            if i not in self.received_packets_list:
                return False
        self.fp.close()
        return True

    def send_data(self, data):
        jsn_data = msgpack.packb(data)
        self.new_socket.sendto(jsn_data, self.client_address)

if __name__ == "__main__":
    server = ThreadingUDPServer(addr, MyRequestHandler)
    server.serve_forever()
