'''
* Autor: Steve Cloud
*
* Create on 2016/10/12
'''

# udpfiletrans
本项目旨在通过udp在两个不同的机器，甚至是网络之间可靠性的传输文件。


{
	'filename':
	'file_md5'
	'file_path'
	'file_packets'
	'packet_md5'
	'packet_index'
	'file_offset'
	'data':''      # 块数据
    'status'
}

status:
    syn-ack         # 握手应答
    exist			# file exist
    block-ack       # 块收包正常
    uncorrect       # 块包不正常
    finished        # 传输成功
    failed          # 传输失败